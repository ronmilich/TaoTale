import type { Item } from "../../api/client";

export type Transcript = NonNullable<Item["transcript"]>;

// The Reader has already validated the stored segments in ascending time order.
export function segmentAt(segments: Transcript["segments"], ms: number) {
  let low = 0;
  let high = segments.length;
  while (low < high) {
    const middle = Math.floor((low + high) / 2);
    const segment = segments[middle];
    if (ms < segment.startMs) high = middle;
    else if (ms >= segment.endMs) low = middle + 1;
    else return middle;
  }
  return -1;
}

export function formatTime(ms: number) {
  const seconds = Math.floor(ms / 1000);
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
}

export function playbackResumeKey(itemId: string, assetId: string) {
  return `taotale.playback.v1.${itemId}.${assetId}`;
}

export function clearPlaybackResume(itemId: string, assetId: string) {
  try {
    localStorage.removeItem(playbackResumeKey(itemId, assetId));
  } catch {
    // Local storage may be unavailable.
  }
}

export function readResume(key: string, durationMs: number) {
  try {
    const ms = Number(localStorage.getItem(key));
    if (Number.isSafeInteger(ms) && ms > 0 && ms < durationMs) return ms;
    localStorage.removeItem(key);
  } catch {
    // Playback also works with local storage disabled.
  }
  return 0;
}

function saveResume(key: string, ms: number, durationMs: number) {
  try {
    const position = Math.round(ms);
    if (position > 0 && position < durationMs) localStorage.setItem(key, String(position));
    else localStorage.removeItem(key);
  } catch {
    // A full or inaccessible store must not interrupt the recording.
  }
}

type PlaybackView = {
  position(ms: number): void;
  segment(index: number): void;
  playing(value: boolean): void;
  buffering(value: boolean): void;
  ready(): void;
  failed(): void;
};

/** Bind one immutable recording. Disposing it releases its listeners, clock and media. */
export function bindPlayback(
  audio: HTMLAudioElement,
  transcript: Transcript,
  key: string,
  view: PlaybackView,
) {
  const { durationMs, segments } = transcript;
  const controller = new AbortController();
  const options = { signal: controller.signal };
  let frame: number | undefined;
  let active = -1;
  let initialized = false;
  let started = false;
  let lastSave = performance.now();

  const currentMs = () => Math.min(durationMs, audio.currentTime * 1000);
  function sample() {
    const ms = currentMs();
    view.position(audio.ended ? durationMs : ms);
    const next = started && !audio.ended ? segmentAt(segments, ms) : -1;
    if (next !== active) {
      active = next;
      view.segment(next);
    }
  }
  function persist() {
    if (initialized && started) saveResume(key, audio.ended ? 0 : currentMs(), durationMs);
    lastSave = performance.now();
  }
  function stopClock() {
    if (frame !== undefined) cancelAnimationFrame(frame);
    frame = undefined;
  }
  function updateProgress() {
    sample();
    if (!audio.paused && performance.now() - lastSave >= 5000) persist();
  }
  function tick() {
    frame = undefined;
    if (controller.signal.aborted || audio.paused || audio.ended || audio.seeking) return;
    updateProgress();
    frame = requestAnimationFrame(tick);
  }
  function startClock() {
    view.buffering(false);
    if (frame === undefined && !audio.paused && !audio.ended && !audio.seeking) tick();
  }
  function seek(ms: number) {
    if (!initialized || !Number.isFinite(ms)) return;
    started = true;
    audio.currentTime = Math.max(0, Math.min(ms / 1000, durationMs / 1000, audio.duration));
    sample();
  }
  function load() {
    if (initialized || !Number.isFinite(audio.duration) || audio.duration <= 0) return;
    initialized = true;
    const resume = readResume(key, Math.min(durationMs, audio.duration * 1000));
    if (resume > 0) seek(resume);
    view.ready();
  }
  function pause() {
    stopClock();
    view.playing(false);
    view.buffering(false);
    sample();
    persist();
  }

  audio.addEventListener("loadedmetadata", load, options);
  audio.addEventListener("durationchange", load, options);
  audio.addEventListener(
    "play",
    () => {
      started = true;
      lastSave = performance.now();
      view.playing(true);
      sample();
    },
    options,
  );
  audio.addEventListener("playing", startClock, options);
  audio.addEventListener(
    "waiting",
    () => {
      stopClock();
      view.buffering(!audio.paused);
    },
    options,
  );
  audio.addEventListener("seeking", stopClock, options);
  audio.addEventListener(
    "seeked",
    () => {
      sample();
      if (audio.paused) persist();
      else if (audio.readyState >= HTMLMediaElement.HAVE_FUTURE_DATA) startClock();
    },
    options,
  );
  audio.addEventListener("timeupdate", updateProgress, options);
  audio.addEventListener("ratechange", sample, options);
  audio.addEventListener("pause", pause, options);
  audio.addEventListener("ended", pause, options);
  audio.addEventListener("error", view.failed, options);
  window.addEventListener("pagehide", persist, options);
  if (audio.error) view.failed();
  else if (audio.readyState >= HTMLMediaElement.HAVE_METADATA) load();

  return {
    seek,
    dispose() {
      controller.abort();
      stopClock();
      persist();
      audio.pause();
      audio.removeAttribute("src");
      audio.load();
    },
  };
}
