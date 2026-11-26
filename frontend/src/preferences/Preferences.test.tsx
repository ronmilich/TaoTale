import { fireEvent, render } from "@solidjs/testing-library";
import { createSignal, flush } from "solid-js";
import { afterEach, expect, it, vi } from "vitest";
import { PreferenceProvider, usePreferences } from "./Preferences";
import { readPreference } from "./storage";

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
  vi.unstubAllGlobals();
});

it("recovers from invalid stored values", () => {
  localStorage.setItem("taotale.preferences.theme", "invalid");
  localStorage.setItem("taotale.preferences.language", "ja-JP");
  expect(readPreference("theme")).toBe("system");
  expect(readPreference("language")).toBe("zh-CN");
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
