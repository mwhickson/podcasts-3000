function waitForBackend(timeout = 10000) {
  const started = Date.now();
  return new Promise((resolve, reject) => {
    const check = () => {
      const backend = globalThis.go?.podcasts?.App;
      if (backend) {
        resolve(backend);
        return;
      }
      if (Date.now() - started >= timeout) {
        reject(new Error("Wails backend bindings were not initialized"));
        return;
      }
      setTimeout(check, 25);
    };
    check();
  });
}

export async function ListPodcasts() {
  return (await waitForBackend()).ListPodcasts();
}
export async function ListEpisodes(podcastID) {
  return (await waitForBackend()).ListEpisodes(podcastID);
}
export async function Subscribe(feedURL) {
  return (await waitForBackend()).Subscribe(feedURL);
}
export async function RefreshAllFeeds() {
  return (await waitForBackend()).RefreshAllFeeds();
}
export async function GetAudioSource(episodeID) {
  return (await waitForBackend()).GetAudioSource(episodeID);
}
export async function SaveProgress(episodeID, seconds, completed) {
  return (await waitForBackend()).SaveProgress(episodeID, seconds, completed);
}
export async function MarkPodcastPlayed(podcastID) {
  return (await waitForBackend()).MarkPodcastPlayed(podcastID);
}
export async function UpdateDuration(episodeID, seconds) {
  return (await waitForBackend()).UpdateDuration(episodeID, seconds);
}
export async function GetLastPlayed() {
  return (await waitForBackend()).GetLastPlayed();
}
export async function SearchITunes(query) {
  return (await waitForBackend()).SearchITunes(query);
}
export async function ImportOPML(path) {
  return (await waitForBackend()).ImportOPML(path);
}
export async function ImportOPMLResults(path) {
  return (await waitForBackend()).ImportOPMLResults(path);
}
export async function ExportOPML(path) {
  return (await waitForBackend()).ExportOPML(path);
}
export async function ImportOPMLDialog() {
  return (await waitForBackend()).ImportOPMLDialog();
}
export async function ExportOPMLDialog() {
  return (await waitForBackend()).ExportOPMLDialog();
}
export async function ToggleFavorite(episodeID) {
  return (await waitForBackend()).ToggleFavorite(episodeID);
}
export async function SaveNote(episodeID, note) {
  return (await waitForBackend()).SaveNote(episodeID, note);
}
export async function PurgeCache(scope, targetID) {
  return (await waitForBackend()).PurgeCache(scope, targetID);
}
export async function Unsubscribe(podcastID) {
  return (await waitForBackend()).Unsubscribe(podcastID);
}
export async function Confirm(message) {
  return (await waitForBackend()).Confirm(message);
}
export async function ShowError(message) {
  return (await waitForBackend()).ShowError(message);
}
export async function ShowInfo(message) {
  return (await waitForBackend()).ShowInfo(message);
}
