import { For, Show, createEffect, createSignal } from "solid-js";
import { apiGet } from "../../api/client";
import controls from "../../components/Controls.module.css";
import styles from "./Settings.module.css";

export default function SystemStatus() {
  const [status, setStatus] = createSignal<Awaited<ReturnType<typeof apiGet<"/system/status">>>>();
  const [failed, setFailed] = createSignal(false);
  const [loading, setLoading] = createSignal(true);
  const [attempt, setAttempt] = createSignal(0);
  createEffect(attempt, () => {
    const request = new AbortController();
    setFailed(false);
    setLoading(true);
    apiGet("/system/status", AbortSignal.any([request.signal, AbortSignal.timeout(10_000)])).then(
      (result) => {
        if (request.signal.aborted) return;
        setStatus(result);
        setLoading(false);
      },
      () => {
        if (request.signal.aborted) return;
        setStatus(undefined);
        setFailed(true);
        setLoading(false);
      },
    );
    return () => request.abort();
  });

  return (
    <>
      <div class={styles.heading}>
        <h2 id="system-status-title">System status</h2>
        <button
          type="button"
          class={controls.secondary}
          disabled={loading()}
          onClick={() => setAttempt((value) => value + 1)}
        >
          {failed() ? "Retry status" : "Refresh status"}
        </button>
      </div>
      <Show when={loading()}>
        <p role="status">Checking system status…</p>
      </Show>
      <Show when={failed()}>
        <p role="alert">Could not reach the local backend.</p>
      </Show>
      <Show when={status()}>
        {(state) => (
          <>
            <dl class={styles.status}>
              <div>
                <dt>PostgreSQL</dt>
                <dd>{state().database === "ready" ? "Ready" : "Unavailable"}</dd>
              </div>
              <div>
                <dt>Database schema</dt>
                <dd>
                  {state().schema === "current" ? "Current" : "Unavailable"}
                  {state().expectedSchemaRevision
                    ? ` · Expected revision ${state().expectedSchemaRevision}`
                    : ""}
                </dd>
              </div>
              <div>
                <dt>Media storage</dt>
                <dd>{state().media === "writable" ? "Writable" : "Unavailable"}</dd>
              </div>
              <div>
                <dt>Azure Speech</dt>
                <dd>
                  {state().speech.azureConfigured ? "Credentials configured" : "Credentials needed"}
                </dd>
              </div>
              <div>
                <dt>Audio generation</dt>
                <dd>
                  {state().speech.provider === "mock"
                    ? state().speech.available
                      ? "Mock provider ready"
                      : "Mock provider unavailable"
                    : state().speech.available
                      ? "Azure runtime ready"
                      : "Azure runtime unavailable"}
                </dd>
              </div>
              <div>
                <dt>App version</dt>
                <dd>{state().appVersion}</dd>
              </div>
              <For each={state().segmentation}>
                {(entry) => (
                  <div>
                    <dt>{entry.languageCode === "zh-CN" ? "Mandarin" : "Korean"} segmentation</dt>
                    <dd>
                      {entry.strategyId} · {entry.version}
                    </dd>
                  </div>
                )}
              </For>
            </dl>
            <p class={styles.hint}>
              Azure verifies credentials and connectivity when you generate audio.
            </p>
          </>
        )}
      </Show>
    </>
  );
}
