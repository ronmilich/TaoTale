// Manual TAO-03 experiment. One request per locale, including in concurrent mode.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
)

type passage struct{ voice, text string }

var cases = map[string]passage{
	"zh-CN": {"zh-CN-XiaoxiaoNeural", "小林说：“你好！”今天有12本书。\n甲 & 乙，3 < 5。😀 明天再读。"},
	"ko-KR": {"ko-KR-SunHiNeural", "지수가 말했어요. “안녕!” 책이 12권 있어요.\n가 & 나, 3 < 5. 😀 내일 읽어요."},
}

var xmlText = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;")

type boundary struct {
	Text                   string
	Type                   common.SpeechSynthesisBoundaryType
	TextOffset, WordLength uint
	AudioTicks             uint64
	DurationTicks          int64
}

type evidence struct {
	Locale, Voice, Rate, Mode, Source, Prefix, SSML string
	Requests                                        int
	Submitted, Finished                             time.Time
	StopTrigger, Reason, Failure                    string
	CancelReason, CancelCode                        string
	DecodedSamples24kHz                             int
	Events                                          []boundary
}

func run() (err error) {
	locale := flag.String("locale", "", "zh-CN, ko-KR, or pair for two concurrent samples")
	rate := flag.String("rate", "1", "0.75, 1 or 1.25")
	mode := flag.String("mode", "sample", "sample, cancel or timeout")
	dir := flag.String("out", "", "new private evidence directory")
	live := flag.Bool("live", false, "submit the selected approved request(s)")
	flag.Parse()

	if !*live {
		return errors.New("live requests require -live")
	}
	if *locale != "pair" {
		return probe(*locale, *rate, *mode, *dir)
	}
	if *mode != "sample" || *dir == "" {
		return errors.New("pair requires sample mode and a new evidence directory")
	}
	if err := os.Mkdir(*dir, 0700); err != nil {
		return err
	}
	results := make(chan error, 2)
	for _, language := range []string{"zh-CN", "ko-KR"} {
		go func() { results <- probe(language, *rate, *mode, filepath.Join(*dir, language)) }()
	}
	return errors.Join(<-results, <-results)
}

func probe(locale, rate, mode, dir string) (err error) {
	p, ok := cases[locale]
	if !ok || dir == "" || (rate != "0.75" && rate != "1" && rate != "1.25") ||
		(mode != "sample" && mode != "cancel" && mode != "timeout") {
		return errors.New("specify -live, a supported locale/rate/mode and a new -out directory")
	}
	key, region := os.Getenv("AZURE_SPEECH_KEY"), os.Getenv("AZURE_SPEECH_REGION")
	if key == "" || region == "" {
		return errors.New("Azure Speech credentials missing")
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		return err
	}
	prefix := fmt.Sprintf(`<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xml:lang="%s"><voice name="%s"><prosody rate="%s">`, locale, p.voice, rate)
	e := evidence{Locale: locale, Voice: p.voice, Rate: rate, Mode: mode, Source: p.text, Prefix: prefix,
		SSML: prefix + xmlText.Replace(p.text) + "</prosody></voice></speak>"}
	var mu sync.Mutex
	save := func() error {
		mu.Lock()
		defer mu.Unlock()
		data, err := json.MarshalIndent(e, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "events.json"), data, 0600)
	}
	defer func() {
		e.Finished = time.Now().UTC()
		if err != nil {
			e.Failure = err.Error()
		}
		err = errors.Join(err, save())
	}()
	config, err := speech.NewSpeechConfigFromSubscription(key, region)
	if err != nil {
		return errors.New("speech config creation failed")
	}
	defer config.Close()
	if err = config.SetSpeechSynthesisOutputFormat(common.Audio24Khz48KBitRateMonoMp3); err != nil {
		return err
	}
	if err = config.SetProperty(common.SpeechServiceResponseRequestWordBoundary, "true"); err != nil {
		return err
	}
	synth, err := speech.NewSpeechSynthesizerFromConfig(config, nil)
	if err != nil {
		return err
	}
	defer synth.Close()
	started := make(chan struct{}, 1)
	synth.SynthesisStarted(func(event speech.SpeechSynthesisEventArgs) {
		defer event.Close()
		select {
		case started <- struct{}{}:
		default:
		}
	})
	synth.WordBoundary(func(event speech.SpeechSynthesisWordBoundaryEventArgs) {
		defer event.Close()
		mu.Lock()
		defer mu.Unlock()
		e.Events = append(e.Events, boundary{event.Text, event.BoundaryType, event.TextOffset, event.WordLength, event.AudioOffset, event.Duration.Nanoseconds() / 100})
	})
	e.Submitted = time.Now().UTC()
	e.Requests = 1
	if err = save(); err != nil {
		return err
	}
	outcomes := synth.SpeakSsmlAsync(e.SSML)
	limit := 30 * time.Second
	if mode == "timeout" {
		limit = 250 * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	var cancelStarted <-chan struct{}
	if mode == "cancel" {
		cancelStarted = started
	}
	var result speech.SpeechSynthesisOutcome
	select {
	case result = <-outcomes:
	case <-cancelStarted:
		e.StopTrigger = "synthesis-started"
	case <-ctx.Done():
		e.StopTrigger = "deadline"
	}
	if e.StopTrigger != "" {
		// The process owns these native handles. Exit on a hung SDK call without
		// freeing handles underneath it. The pre-submission journal survives.
		timer := time.AfterFunc(8*time.Second, func() { fmt.Fprintln(os.Stderr, "SDK stop did not settle"); os.Exit(2) })
		defer timer.Stop()
		stopErr := <-synth.StopSpeakingAsync()
		result = <-outcomes
		defer result.Close()
		if stopErr != nil {
			return errors.New("SDK stop failed")
		}
	} else {
		defer result.Close()
	}
	if result.Result == nil || result.Error != nil {
		return errors.New("SDK returned no usable result")
	}
	e.Reason = result.Result.Reason.String()
	if result.Result.Reason == common.Canceled {
		details, detailErr := speech.NewCancellationDetailsFromSpeechSynthesisResult(result.Result)
		if detailErr != nil {
			return detailErr
		}
		e.CancelReason = details.Reason.String()
		e.CancelCode = details.ErrorCode.String()
	}
	if mode != "sample" {
		if e.StopTrigger == "" || (mode == "cancel" && e.StopTrigger != "synthesis-started") || result.Result.Reason != common.Canceled || e.CancelReason != "CancelledByUser" || e.CancelCode != "NoError" {
			return errors.New("intended cancellation was not observed")
		}
		return nil
	}
	if e.StopTrigger != "" || result.Result.Reason != common.SynthesizingAudioCompleted || len(result.Result.AudioData) == 0 {
		return errors.New("audio synthesis did not complete")
	}
	file := filepath.Join(dir, "audio.mp3")
	if err := os.WriteFile(file, result.Result.AudioData, 0600); err != nil {
		return err
	}
	pcm, err := exec.Command("ffmpeg", "-v", "error", "-xerror", "-f", "mp3", "-i", file, "-ac", "1", "-ar", "24000", "-f", "s16le", "pipe:1").Output()
	if err != nil || len(pcm) < 2 {
		return errors.New("persisted MP3 did not decode")
	}
	e.DecodedSamples24kHz = len(pcm) / 2
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
