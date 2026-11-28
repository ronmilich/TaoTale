import { fireEvent, render } from "@solidjs/testing-library";
import { createSignal, flush } from "solid-js";
import { afterEach, expect, it, vi } from "vitest";
import { PreferenceProvider, usePreferences } from "./Preferences";
import { readPreference, readReaderPreferences, readSpeechPreference } from "./storage";

afterEach(() => {
  vi.restoreAllMocks();
  localStorage.clear();
  delete document.documentElement.dataset.theme;
  vi.unstubAllGlobals();
});

it("recovers from invalid stored values", () => {
  localStorage.setItem("taotale.preferences.theme", "invalid");
  localStorage.setItem("taotale.preferences.language", "ja-JP");
  expect(readPreference("theme")).toBe("system");
  expect(readPreference("language")).toBe("zh-CN");
  localStorage.setItem("taotale.preferences.playbackSpeed", "Infinity");
  localStorage.setItem(
    "taotale.preferences.reader",
    '{"vocabularyDisplay":"unknown","showVocabularyDetails":"true"}',
  );
  localStorage.setItem("taotale.preferences.speech.zh-CN", "{");
  localStorage.setItem(
    "taotale.preferences.speech.ko-KR",
    '{"voiceId":"unknown","speakingRate":7}',
  );
  expect(readPreference("playbackSpeed")).toBe("1");
  expect(readReaderPreferences()).toEqual({
    vocabularyDisplay: "both",
    showVocabularyDetails: false,
  });
  expect(readSpeechPreference("zh-CN")).toBeUndefined();
  expect(readSpeechPreference("ko-KR")).toBeUndefined();
  localStorage.setItem("taotale.preferences.reader", '{"vocabularyDisplay":"acquired"}');
  expect(readReaderPreferences()).toEqual({
    vocabularyDisplay: "acquired",
    showVocabularyDetails: false,
  });
});

it("keeps preferences usable in memory when browser storage is blocked", () => {
  vi.stubGlobal("fetch", () => new Promise(() => {}));
  const denied = () => {
    throw new DOMException("Storage blocked", "SecurityError");
  };
  vi.spyOn(Storage.prototype, "getItem").mockImplementation(denied);
  vi.spyOn(Storage.prototype, "setItem").mockImplementation(denied);
  expect(readPreference("playbackSpeed")).toBe("1");
  expect(readReaderPreferences().showVocabularyDetails).toBe(false);
  expect(readSpeechPreference("ko-KR")).toBeUndefined();
  function Settings() {
    const preferences = usePreferences();
    return (
      <button
        type="button"
        onClick={() => {
          preferences.changePlaybackSpeed("1.5");
          preferences.changeReader({ showVocabularyDetails: true });
          preferences.changeSpeech("ko-KR", { voiceId: "ko-KR-mock", speakingRate: 1.25 });
        }}
      >
        {preferences.playbackSpeed()} {String(preferences.reader().showVocabularyDetails)}{" "}
        {preferences.speech("ko-KR")?.speakingRate}
      </button>
    );
  }
  const view = render(() => (
    <PreferenceProvider>
      <Settings />
    </PreferenceProvider>
  ));
  fireEvent.click(view.getByRole("button"));
  flush();
  expect(view.getByRole("button")).toHaveTextContent("1.5 true 1.25");
});

it("applies and saves a theme without remounting a stateful child", () => {
  vi.stubGlobal("fetch", () => new Promise(() => {}));
  let mounts = 0;
  function Stateful() {
    mounts++;
    const preferences = usePreferences();
    const [count, setCount] = createSignal(0);
    return (
      <>
        <button type="button" onClick={() => setCount((value) => value + 1)}>
          Count {count()}
        </button>
        <button type="button" onClick={() => preferences.changeTheme("dark")}>
          Use dark
        </button>
      </>
    );
  }
  const view = render(() => (
    <PreferenceProvider>
      <Stateful />
    </PreferenceProvider>
  ));
  const counter = view.getByRole("button", { name: "Count 0" });
  fireEvent.click(counter);
  flush();
  fireEvent.click(view.getByRole("button", { name: "Use dark" }));
  flush();
  expect(view.getByRole("button", { name: "Count 1" })).toBe(counter);
  expect(document.documentElement.dataset.theme).toBe("dark");
  expect(localStorage.getItem("taotale.preferences.theme")).toBe("dark");
  expect(mounts).toBe(1);
});
