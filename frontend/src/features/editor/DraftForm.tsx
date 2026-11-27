import { For, Show, createMemo, type ParentProps } from "solid-js";
import type { LearningLanguage } from "../../api/client";
import controls from "../../components/Controls.module.css";
import TopicPicker from "../topics/TopicPicker";
import type { useTopics } from "../topics/topics";
import {
  applyTextareaEdit,
  difficulties,
  displayedText,
  replaceTextareaRange,
  textLength,
  textLimits,
  type DraftFields,
  type TextField,
} from "./draft";
import styles from "./DraftEditor.module.css";

type DraftFormProps = {
  values: DraftFields;
  errors: Partial<Record<keyof DraftFields, string>>;
  languages: LearningLanguage[] | undefined;
  languageError: boolean;
  retryLanguages: () => void;
  saved: boolean;
  revisionHref?: string;
  saving: boolean;
  audio: ParentProps["children"];
  cover: ParentProps["children"];
  topics: ReturnType<typeof useTopics>;
  update: <K extends keyof DraftFields>(name: K, value: DraftFields[K]) => void;
};

export function DraftForm(props: DraftFormProps) {
  const languageName = createMemo(() => {
    const code = props.values.languageCode;
    return (
      props.languages?.find((language) => language.code === code)?.displayName ??
      (code === "ko-KR" ? "Korean" : "Mandarin Chinese")
    );
  });

  return (
    <fieldset class={styles.fields} disabled={props.saving}>
      <div class={styles.columns}>
        <div class={styles.writing}>
          <section class={styles.paper} aria-label="Article text">
            <div class={styles.field}>
              <label for="draft-languageCode">Item language</label>
              <Show
                when={!props.saved}
                fallback={<input id="draft-languageCode" value={languageName()} readonly />}
              >
                <select
                  id="draft-languageCode"
                  value={props.values.languageCode}
                  disabled={!props.languages?.length}
                  aria-invalid={props.errors.languageCode ? "true" : undefined}
                  aria-describedby="draft-languageCode-message"
                  onChange={(event) =>
                    props.update(
                      "languageCode",
                      event.currentTarget.value as DraftFields["languageCode"],
                    )
                  }
                >
                  <option value="">Choose a language</option>
                  <For each={props.languages}>
                    {(entry) => (
                      <option
                        value={entry.code}
                        selected={entry.code === props.values.languageCode}
                      >
                        {entry.displayName}
                      </option>
                    )}
                  </For>
                </select>
                <p
                  id="draft-languageCode-message"
                  class={props.errors.languageCode ? styles.fieldError : styles.hint}
                >
                  {props.errors.languageCode ??
                    (props.languageError
                      ? "Item languages are unavailable."
                      : props.languages === undefined
                        ? "Loading item languages…"
                        : props.languages.length === 0
                          ? "No enabled item languages."
                          : "")}
                </p>
                <Show when={props.languageError}>
                  <button
                    type="button"
                    class={`${controls.secondary} ${styles.fieldAction}`}
                    onClick={() => props.retryLanguages()}
                  >
                    Retry languages
                  </button>
                </Show>
              </Show>
            </div>
            <DraftText
              name="title"
              label="Title"
              value={props.values.title}
              error={props.errors.title}
              update={(value) => props.update("title", value)}
            />
            <DraftText
              name="sourceText"
              label="Source text"
              rows={9}
              lang={props.values.languageCode}
              value={props.values.sourceText}
              error={props.errors.sourceText}
              readOnly={props.revisionHref !== undefined}
              update={(value) => props.update("sourceText", value)}
            />
          </section>
          <Show when={props.revisionHref} fallback={props.audio}>
            {(href) => (
              <section class={styles.paper} aria-label="Published recording">
                <div class={styles.speechFields}>
                  <div class={styles.field}>
                    <label for="draft-voiceId">Voice</label>
                    <input
                      id="draft-voiceId"
                      value={props.values.voiceId ?? "Unavailable"}
                      readonly
                    />
                  </div>
                  <div class={styles.field}>
                    <label for="draft-speakingRate">Speaking rate</label>
                    <input
                      id="draft-speakingRate"
                      value={
                        props.values.speakingRate ? `${props.values.speakingRate}×` : "Unavailable"
                      }
                      readonly
                    />
                  </div>
                </div>
                <a class={[controls.secondary, styles.fieldAction]} href={href()}>
                  Edit Text &amp; Voice
                </a>
              </section>
            )}
          </Show>
        </div>
        <section class={styles.details} aria-labelledby="draft-details">
          <h2 id="draft-details">Article details</h2>
          {props.cover}
          <TopicPicker
            topics={props.topics}
            selected={props.values.topicIds}
            change={(ids) => props.update("topicIds", ids)}
          />
          <div class={styles.field}>
            <label for="draft-difficulty">Difficulty</label>
            <select
              id="draft-difficulty"
              value={props.values.difficulty}
              onChange={(event) =>
                props.update("difficulty", event.currentTarget.value as DraftFields["difficulty"])
              }
            >
              <For each={difficulties}>
                {(entry) => (
                  <option value={entry.value} selected={entry.value === props.values.difficulty}>
                    {entry.label}
                  </option>
                )}
              </For>
            </select>
          </div>
          <DraftText
            name="description"
            label="Description"
            rows={3}
            value={props.values.description}
            error={props.errors.description}
            update={(value) => props.update("description", value)}
          />
          <DraftText
            name="notes"
            label="Notes"
            rows={4}
            value={props.values.notes}
            error={props.errors.notes}
            update={(value) => props.update("notes", value)}
          />
        </section>
      </div>
    </fieldset>
  );
}

function DraftText(props: {
  name: TextField;
  label: string;
  rows?: number;
  lang?: string;
  value: string;
  error?: string;
  readOnly?: boolean;
  update: (value: string) => void;
}) {
  const id = () => `draft-${props.name}`;
  const isSource = () => props.name === "sourceText";
  const countId = () => `${id()}-count`;
  const errorId = () => `${id()}-error`;
  const descriptions = () => (props.error ? `${countId()} ${errorId()}` : countId());
  const count = createMemo(() => textLength(props.name, props.value));
  let paste: string | undefined;

  function rememberPaste(event: ClipboardEvent & { currentTarget: HTMLTextAreaElement }) {
    const inserted = event.clipboardData?.getData("text/plain");
    paste = inserted
      ? replaceTextareaRange(
          props.value,
          event.currentTarget.selectionStart,
          event.currentTarget.selectionEnd,
          inserted,
        )
      : undefined;
  }

  function editTextarea(event: InputEvent & { currentTarget: HTMLTextAreaElement }) {
    const next = event.currentTarget.value;
    props.update(
      event.inputType === "insertFromPaste" && paste !== undefined && displayedText(paste) === next
        ? paste
        : applyTextareaEdit(props.value, next),
    );
    paste = undefined;
  }

  return (
    <div class={styles.field}>
      <div class={styles.fieldHead}>
        <label for={id()}>{props.label}</label>
        <Show when={!props.readOnly}>
          <span
            id={countId()}
            class={count() > textLimits[props.name] ? styles.fieldError : styles.count}
          >
            {count()} / {textLimits[props.name]}
            {isSource() ? " UTF-16 units" : " characters"}
          </span>
        </Show>
      </div>
      <Show
        when={props.rows}
        fallback={
          <input
            id={id()}
            type="text"
            value={props.value}
            aria-invalid={props.error ? "true" : undefined}
            aria-describedby={descriptions()}
            onInput={(event) => props.update(event.currentTarget.value)}
          />
        }
      >
        <textarea
          id={id()}
          class={isSource() ? styles.source : undefined}
          rows={props.rows}
          lang={props.lang || undefined}
          readonly={props.readOnly}
          spellcheck={isSource() ? "false" : "true"}
          value={displayedText(props.value)}
          aria-invalid={props.error ? "true" : undefined}
          aria-describedby={props.readOnly ? undefined : descriptions()}
          onPaste={rememberPaste}
          onInput={editTextarea}
        />
      </Show>
      <Show when={props.error}>
        <p class={styles.fieldError} id={errorId()}>
          {props.error}
        </p>
      </Show>
    </div>
  );
}
