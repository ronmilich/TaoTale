import { Show, createSignal } from "solid-js";
import controls from "../../components/Controls.module.css";
import styles from "./Settings.module.css";

const backup = `$backup = Join-Path ([Environment]::GetFolderPath('MyDocuments')) ("TaoTale-" + (Get-Date -Format yyyyMMdd-HHmmss))
New-Item -ItemType Directory -Path $backup -ErrorAction Stop | Out-Null
docker volume inspect taotale-backend_media | Out-Null
if ($LASTEXITCODE) { throw "Could not find the media volume." }
docker compose -p taotale-backend stop api
if ($LASTEXITCODE) { throw "Could not stop the API." }
try {
    docker compose -p taotale-backend exec -T db pg_dump -U taotale -Fc -f /tmp/taotale.dump taotale
    if ($LASTEXITCODE) { throw "The database dump failed." }
    docker compose -p taotale-backend cp db:/tmp/taotale.dump "$backup/database.dump"
    if ($LASTEXITCODE) { throw "Could not copy the database dump." }
    docker run --rm --mount "type=volume,source=taotale-backend_media,target=/media,readonly" --mount "type=bind,source=$backup,target=/backup" --entrypoint tar postgres:17.11 -czf /backup/media.tar.gz -C /media .
    if ($LASTEXITCODE) { throw "The media archive failed." }
} finally {
    docker compose -p taotale-backend start api
    if ($LASTEXITCODE) { Write-Warning "Could not restart the API. Start it manually before using TaoTale." }
}
$backup`;

const restore = `if (!$backup -or !(Test-Path -LiteralPath "$backup/database.dump" -PathType Leaf) -or !(Test-Path -LiteralPath "$backup/media.tar.gz" -PathType Leaf)) {
    throw "Set the backup folder to a directory containing database.dump and media.tar.gz."
}
$restore = 'taotale-restore-' + [guid]::NewGuid().ToString('N').Substring(0, 8)
$mediaVolume = $restore + '_media'
$priorPort = $env:PORT
$env:PORT = '18081'
try {
    docker compose -p $restore up --wait db
    if ($LASTEXITCODE) { throw "Could not start the restore database." }
    docker compose -p $restore cp "$backup/database.dump" db:/tmp/taotale.dump
    if ($LASTEXITCODE) { throw "Could not copy the database dump." }
    docker compose -p $restore exec -T db pg_restore -U taotale -d taotale --exit-on-error /tmp/taotale.dump
    if ($LASTEXITCODE) { throw "The database restore failed." }
    docker volume create --label "com.docker.compose.project=$restore" --label "com.docker.compose.volume=media" $mediaVolume
    if ($LASTEXITCODE) { throw "Could not create the restore media volume." }
    docker run --rm --mount "type=volume,source=$mediaVolume,target=/media" --mount "type=bind,source=$backup,target=/backup,readonly" --entrypoint tar postgres:17.11 -xzf /backup/media.tar.gz -C /media
    if ($LASTEXITCODE) { throw "The media restore failed." }
    docker compose -p $restore up --build --wait
    if ($LASTEXITCODE) { throw "Could not start the restored app." }
} finally {
    $env:PORT = $priorPort
}
$restore`;

function Commands(props: { label: string; value: string }) {
  const [notice, setNotice] = createSignal("");
  const [failed, setFailed] = createSignal(false);
  async function copy() {
    try {
      await navigator.clipboard.writeText(props.value);
      setFailed(false);
      setNotice("Commands copied.");
    } catch {
      setFailed(true);
      setNotice("Clipboard unavailable. Select and copy the commands below.");
    }
  }
  return (
    <>
      <button type="button" class={controls.secondary} onClick={() => void copy()}>
        Copy {props.label} commands
      </button>
      <Show when={notice()}>
        <p role={failed() ? "alert" : "status"}>{notice()}</p>
      </Show>
      <pre tabindex={0} class={styles.commands} aria-label={`${props.label} commands`}>
        <code>{props.value}</code>
      </pre>
    </>
  );
}

export default function LocalData() {
  return (
    <>
      <h2 id="local-data-title">Local data</h2>
      <dl class={styles.status}>
        <div>
          <dt>Articles and vocabulary</dt>
          <dd>PostgreSQL Docker volume</dd>
        </div>
        <div>
          <dt>Recordings and covers</dt>
          <dd>
            Media Docker volume at <code>/var/lib/taotale/media</code> inside the container
          </dd>
        </div>
        <div>
          <dt>Preferences</dt>
          <dd>This browser profile</dd>
        </div>
      </dl>
      <p>
        Keep the database dump and media archive from the same backup together. Keep your browser
        profile and local setup credentials separately.
      </p>
      <details>
        <summary>Back up the local installation</summary>
        <p>
          With the default Compose stack running, run these commands in PowerShell from{" "}
          <code>infra</code>. The backup folder is created in Documents. The API pauses during the
          backup and starts again afterwards. If it fails to start, run{" "}
          <code>docker compose -p taotale-backend start api</code>.
        </p>
        <Commands label="backup" value={backup} />
      </details>
      <details>
        <summary>Restore a backup into a fresh installation</summary>
        <p>
          Set <code>$backup</code> to the folder containing both files. Run these commands from{" "}
          <code>infra</code> with port 18081 free. The commands use a separate Compose project and
          new data volumes.
        </p>
        <Commands label="restore" value={restore} />
        <p>
          Open the restored Library and check articles, vocabulary, covers, and playback at{" "}
          <a href="http://127.0.0.1:18081/library">127.0.0.1:18081</a>. Stop the restored stack with{" "}
          <code>docker compose -p $restore stop</code>.
        </p>
      </details>
    </>
  );
}
