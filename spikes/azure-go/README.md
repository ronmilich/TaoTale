# TAO-03 Azure Go feasibility probe

The two fixed synthetic passages exercise Mandarin and Korean punctuation, quotation marks, numbers, newlines, XML entities and an emoji. Each probe writes its submitted SSML, SDK result, timestamps and raw word-boundary callbacks to a new private directory. A completed sample also writes the MP3 and decodes it to mono 24 kHz PCM with FFmpeg to measure duration.

## Build and run

From this directory, build with `docker build --platform linux/amd64 -t taotale-azure-spike .`. The Dockerfile installs Go 1.27.4, the v1.53.0 Go binding and matching Microsoft native library, a C toolchain, ALSA and FFmpeg on Debian Bookworm. The Go binding needs CGO, the C API headers and a dynamic loader that can find the native shared libraries. The current static backend image needs a runtime change before adopting this adapter. [Microsoft SDK setup](https://learn.microsoft.com/azure/ai-services/speech-service/quickstarts/setup-platform?pivots=programming-language-go)

Live runs require explicit spend approval. Set `AZURE_SPEECH_KEY` and `AZURE_SPEECH_REGION` in the local environment and mount an existing evidence directory outside the repository. For example, from PowerShell with an absolute path in `$evidence`, this submits **one request**.

```powershell
docker run --rm -e AZURE_SPEECH_KEY -e AZURE_SPEECH_REGION --mount "type=bind,source=$evidence,target=/evidence" taotale-azure-spike -live -locale zh-CN -rate 0.75 -out /evidence/zh-CN-0.75
```

For the eight-request experiment, run `-locale pair -rate 1` once for the two concurrent samples, then each locale at 0.75 and 1.25. Use `-mode cancel` for Mandarin at 1 and `-mode timeout` for Korean at 1. Use a new output directory for each invocation. The cancellation probe stops on the SDK start event. The timeout probe stops after 250 ms. Each awaits the outstanding result and bounds shutdown time.

Inspect existing captures offline with Node. This reads evidence from the mounted directory and never contacts Azure.

```powershell
$captures = @(Get-ChildItem -LiteralPath $evidence -Recurse -Filter events.json | ForEach-Object FullName)
node check.mjs @captures
```

## Results on 25 November 2026

| Voice | Decoded MP3 duration at 0.75 / 1 / 1.25 | Boundary callbacks per sample | Decision |
|---|---|---|---|
| zh-CN-XiaoxiaoNeural | 11.232 / 8.424 / 6.768 seconds | 15 word, 6 punctuation | Go for adapter development |
| ko-KR-SunHiNeural | 13.248 / 9.936 / 7.968 seconds | 14 word, 7 punctuation | Go for adapter development |

All six recordings decoded and all 126 callbacks matched the exact escaped text at their reported offsets. In this SDK/runtime, both `TextOffset` and `WordLength` count Unicode code points in the full escaped SSML. The emoji occupied one provider position and two UTF-16 units. Entity text included its XML spelling. Build the offset map during encoding, then translate both endpoints back to original UTF-16. Validate raw 100 ns ticks before rounding to milliseconds. Word timings were positive, ordered and nonoverlapping, all speech letters and digits were covered, and every event fit its decoded recording. Korean produced a zero-duration punctuation event for the less-than sign. Punctuation supplies no speech timing.

The two independent synthesizers ran in one Go process at rate 1. Start-triggered cancellation and deadline cancellation each settled with `CancelledByUser` and `NoError`. The historical Node Mandarin smoke produced a 5.904-second READY recording after two earlier alignment failures. It used different text and recorded segment-level evidence. Both implementations produced Mandarin MP3 with timing. The Go captures additionally establish the offset unit after a supplementary character. [Legacy smoke](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/development/evidence/m4/azure-smoke.json)

The private evidence folder is `TaoTale Evidence/TAO-03-20261125` under Documents. Eight explicit requests were submitted with zero application retries. Counting escaped content, prosody markup and double-weighted Han characters gives 708 estimated units including both stops. At the regional public S1 rate of USD 15 per million, the retail estimate is **USD 0.01062**. Actual invoiced cost depends on metering of canceled requests, account pricing and free allowance. [Billing rules](https://learn.microsoft.com/azure/ai-services/speech-service/text-to-speech#billable-characters) · [Azure pricing](https://azure.microsoft.com/en-us/pricing/details/speech/)
