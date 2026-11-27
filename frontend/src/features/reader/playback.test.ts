import { afterEach, expect, it, vi } from "vitest";
import { bindPlayback, readResume, segmentAt, type Transcript } from "./playback";

const timing: Transcript = {
  durationMs: 15000,
  segments: [
    { index: 0, textStart: 0, textEnd: 1, text: "茶", startMs: 100, endMs: 6000 },
    { index: 1, textStart: 2, textEnd: 3, text: "水", startMs: 7000, endMs: 14000 },
  ],
};
afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  localStorage.clear();
});

it("uses half-open timings and leaves leading, internal and trailing gaps unhighlighted", () => {
  for (const [ms, index] of [
    [0, -1],
    [100, 0],
    [5999, 0],
    [6000, -1],
    [6999, -1],
    [7000, 1],
    [14000, -1],
    [15000, -1],
  ]) {
    expect(segmentAt(timing.segments, ms)).toBe(index);
  }
});

it("samples only during playback, writes resume periodically, and releases all listeners on disposal", () => {
  const audio = document.createElement("audio");
  const state = { paused: true, ended: false };
  Object.defineProperties(audio, {
    readyState: { value: 4 },
    duration: { value: 15 },
    paused: { get: () => state.paused },
    ended: { get: () => state.ended },
  });
  vi.spyOn(audio, "pause").mockImplementation(() => {
    state.paused = true;
  });
  vi.spyOn(audio, "load").mockImplementation(() => {});
  const frames = new Map<number, FrameRequestCallback>();
  let nextID = 0;
  vi.stubGlobal("requestAnimationFrame", (fn: FrameRequestCallback) => {
    frames.set(++nextID, fn);
    return nextID;
  });
  vi.stubGlobal("cancelAnimationFrame", (id: number) => frames.delete(id));
  let now = 0;
  vi.spyOn(performance, "now").mockImplementation(() => now);
  function tick(ms: number) {
    expect(frames.size).toBe(1);
    const [id, frame] = [...frames][0];
    frames.delete(id);
    now = ms;
    audio.currentTime = ms / 1000;
    frame(now);
  }
  const view = {
    position: vi.fn(),
    segment: vi.fn(),
    playing: vi.fn(),
    buffering: vi.fn(),
    ready: vi.fn(),
    failed: vi.fn(),
  };
  const key = "taotale.playback.v1.item.asset";
  const binding = bindPlayback(audio, timing, key, view);
  const fire = (name: string) => audio.dispatchEvent(new Event(name));
  expect(frames.size).toBe(0);
  state.paused = false;
  fire("play");
  fire("playing");
  fire("playing");
  for (let ms = 200; ms < 5000; ms += 200) tick(ms);
  expect(localStorage.getItem(key)).toBe(null);
  tick(5000);
  expect(readResume(key, 15000)).toBe(5000);
  expect(view.segment.mock.calls).toEqual([[0]]);
  tick(6500);
  tick(7200);
  expect(view.segment.mock.calls).toEqual([[0], [-1], [1]]);
  fire("waiting");
  expect(frames.size).toBe(0);
  fire("playing");
  state.paused = true;
  fire("pause");
  expect(frames.size).toBe(0);
  expect(readResume(key, 15000)).toBe(7200);
  binding.seek(12000);
  expect(audio.currentTime).toBe(12);
  state.paused = false;
  fire("play");
  fire("playing");
  binding.dispose();
  expect(frames.size).toBe(0);
  expect(audio.paused).toBe(true);
  expect(readResume(key, 15000)).toBe(12000);
  audio.currentTime = 1;
  const samples = view.position.mock.calls.length;
  state.paused = false;
  for (const event of ["play", "playing", "pause", "seeked", "timeupdate", "ended", "error"])
    fire(event);
  window.dispatchEvent(new Event("pagehide"));
  expect(view.position).toHaveBeenCalledTimes(samples);
  expect(view.failed).not.toHaveBeenCalled();
  expect(frames.size).toBe(0);
  expect(readResume(key, 15000)).toBe(12000);
});
