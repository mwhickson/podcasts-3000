import * as App from "./wailsjs/go/main/App.js";

const state = { podcast: null, episodes: [] };
const $ = (selector) => document.querySelector(selector);
const escapeHTML = (value) => {
  const element = document.createElement("div");
  element.textContent = value || "";
  return element.innerHTML;
};

async function loadPodcasts() {
  const podcasts = await App.ListPodcasts();
  $("#podcasts").innerHTML = podcasts.map((podcast) =>
    `<div class="podcast ${state.podcast?.id === podcast.id ? "active" : ""}" data-id="${podcast.id}">${escapeHTML(podcast.title)}</div>`
  ).join("");
  document.querySelectorAll(".podcast").forEach((element) => {
    element.onclick = () => selectPodcast(Number(element.dataset.id));
  });
  if (!state.podcast && podcasts.length) await selectPodcast(podcasts[0].id);
}

async function selectPodcast(id) {
  const podcasts = await App.ListPodcasts();
  state.podcast = podcasts.find((podcast) => podcast.id === id);
  state.episodes = await App.ListEpisodes(id);
  $("#episodes").innerHTML = state.episodes.length ? state.episodes.map((episode) =>
    `<article class="episode"><div><h3>${episode.isFavorite ? "★ " : ""}${escapeHTML(episode.title)}</h3><p>${escapeHTML(episode.description)}</p></div><button data-episode="${episode.id}">Play</button></article>`
  ).join("") : `<p class="muted">No episodes found in this feed.</p>`;
  document.querySelectorAll("[data-episode]").forEach((button) => {
    button.onclick = () => play(Number(button.dataset.episode));
  });
  await loadPodcasts();
}

async function play(id) {
  const episode = state.episodes.find((item) => item.id === id);
  const source = await App.GetAudioSource(id);
  $("#audio").src = source;
  $("#now-playing").textContent = episode.title;
  $("#audio").play();
}

$("#subscribe").onsubmit = async (event) => {
  event.preventDefault();
  try {
    await App.Subscribe($("#feed-url").value);
    $("#feed-url").value = "";
    await loadPodcasts();
  } catch (error) {
    alert(error);
  }
};

$("#refresh-all").onclick = async () => {
  await App.RefreshAllFeeds();
  await loadPodcasts();
  if (state.podcast) await selectPodcast(state.podcast.id);
};

$("#audio").addEventListener("timeupdate", () => {
  const episode = state.episodes.find((item) => item.title === $("#now-playing").textContent);
  if (episode && Math.floor($("#audio").currentTime) % 5 === 0) {
    App.SaveProgress(episode.id, Math.floor($("#audio").currentTime), false);
  }
});
$("#audio").addEventListener("ended", () => {
  const episode = state.episodes.find((item) => item.title === $("#now-playing").textContent);
  if (episode) App.SaveProgress(episode.id, Math.floor($("#audio").duration), true);
});

loadPodcasts().catch((error) => { $("#episodes").innerHTML = `<p class="muted">${escapeHTML(String(error))}</p>`; });
