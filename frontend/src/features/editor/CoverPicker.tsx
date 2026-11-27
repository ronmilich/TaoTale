import { Show, createSignal, onCleanup } from "solid-js";
import { ApiError, coverURL, uploadCover } from "../../api/client";
import controls from "../../components/Controls.module.css";
import styles from "./DraftEditor.module.css";

const maxCoverBytes = 1_000_000;
const maxCoverDimension = 4096;
const formats = "Choose a JPEG, PNG or WebP image.";

async function coverType(file: File) {
  const bytes = new Uint8Array(await file.slice(0, 12).arrayBuffer());
  if (bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff) return "image/jpeg";
  if ([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a].every((byte, i) => bytes[i] === byte))
    return "image/png";
  const signature = String.fromCharCode(...bytes);
  if (signature.startsWith("RIFF") && signature.slice(8) === "WEBP") return "image/webp";
}

async function coverDimensions(file: File, contentType: string) {
  const url = URL.createObjectURL(file.slice(0, file.size, contentType));
  try {
    const image = new Image();
    image.src = url;
    await image.decode();
    return { width: image.naturalWidth, height: image.naturalHeight };
  } finally {
    URL.revokeObjectURL(url);
  }
}

export default function CoverPicker(props: {
  imageId: string | null;
  uploading: boolean;
  setUploading: (uploading: boolean) => void;
  change: (id: string | null) => void;
}) {
  const [error, setError] = createSignal("");
  const [retryFile, setRetryFile] = createSignal<File>();
  const [failedImage, setFailedImage] = createSignal<string>();
  let upload: AbortController | undefined;
  onCleanup(() => upload?.abort());

  async function select(file: File) {
    if (upload) return;
    setError("");
    setRetryFile(undefined);
    if (file.size > maxCoverBytes) {
      setError("Cover images must be 1,000,000 bytes or smaller.");
      return;
    }

    const controller = new AbortController();
    upload = controller;
    props.setUploading(true);
    try {
      const contentType = await coverType(file);
      if (controller.signal.aborted) return;
      if (!contentType) {
        setError(formats);
        return;
      }
      const dimensions = await coverDimensions(file, contentType).catch(() => undefined);
      if (controller.signal.aborted) return;
      if (!dimensions?.width || !dimensions.height) {
        setError("This image could not be read. " + formats);
        return;
      }
      if (dimensions.width > maxCoverDimension || dimensions.height > maxCoverDimension) {
        setError("Cover images must be at most 4096 × 4096 pixels.");
        return;
      }
      const image = await uploadCover(file, contentType, controller.signal);
      if (!controller.signal.aborted) props.change(image.id);
    } catch (cause) {
      if (controller.signal.aborted) return;
      const rejected = cause instanceof ApiError && cause.status >= 400 && cause.status < 500;
      setError(rejected ? cause.message : "Could not upload the cover. Try again.");
      if (!rejected) setRetryFile(file);
    } finally {
      upload = undefined;
      if (!controller.signal.aborted) props.setUploading(false);
    }
  }

  return (
    <div class={styles.field}>
      <label for="draft-coverImageId">Cover image</label>
      <div class={styles.coverPreview}>
        <Show
          when={props.imageId === failedImage() ? undefined : props.imageId}
          keyed
          fallback={<span>{props.imageId ? "Cover unavailable" : "No cover selected"}</span>}
        >
          {(id) => (
            <img src={coverURL(id)} alt="Cover preview" onError={() => setFailedImage(id)} />
          )}
        </Show>
      </div>
      <input
        id="draft-coverImageId"
        type="file"
        accept="image/jpeg,image/png,image/webp"
        disabled={props.uploading}
        aria-invalid={error() ? "true" : undefined}
        aria-describedby={error() ? "cover-limits cover-error" : "cover-limits"}
        onChange={(event) => {
          const input = event.currentTarget;
          const file = input.files?.[0];
          input.value = "";
          if (file) void select(file);
        }}
      />
      <p id="cover-limits" class={styles.hint}>
        JPEG, PNG or WebP. Up to 1,000,000 bytes and 4096 × 4096 pixels.
      </p>
      <Show when={props.uploading}>
        <p role="status">Checking and uploading cover…</p>
      </Show>
      <Show when={error()}>
        <p id="cover-error" class={styles.fieldError} role="alert">
          {error()}
        </p>
      </Show>
      <div class={styles.coverActions}>
        <Show when={retryFile()} keyed>
          {(file) => (
            <button type="button" class={controls.secondary} onClick={() => void select(file)}>
              Retry cover upload
            </button>
          )}
        </Show>
        <Show when={props.imageId}>
          <button
            type="button"
            class={controls.secondary}
            disabled={props.uploading}
            onClick={() => {
              setError("");
              setRetryFile(undefined);
              props.change(null);
            }}
          >
            Remove cover
          </button>
        </Show>
      </div>
    </div>
  );
}
