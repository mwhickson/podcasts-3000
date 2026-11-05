import * as App from "./wailsjs/go/podcasts/App.js";

const state = { podcast: null, episodes: [], episode: null };
const $ = (selector) => document.querySelector(selector);
const escapeHTML = (value) => {
  const element = document.createElement("div");
  element.textContent = value || "";
  return element.innerHTML;
};
const messageText = (error) => error instanceof Error ? error.message : String(error);
const showError = (error) => App.ShowError(messageText(error)).catch(console.error);
const confirmAction = (message) => App.Confirm(message).catch((error) => {
  console.error(error);
  showError(error);
  return false;
});

async function loadPodcasts() {
  const podcasts = (await App.ListPodcasts()) || [];
  $("#podcasts").innerHTML = podcasts.map((podcast) =>
    `<div class="podcast ${state.podcast?.id === podcast.id ? "active" : ""}" data-id="${podcast.id}"><span class="podcast-name">${escapeHTML(podcast.title)}</span><span class="podcast-actions"><button class="compact-button" data-purge-podcast="${podcast.id}" title="Purge this podcast's downloads">Purge</button><button class="compact-button" data-unsubscribe="${podcast.id}" title="Unsubscribe from this podcast">Unsubscribe</button></span></div>`
  ).join("");
  document.querySelectorAll(".podcast").forEach((element) => {
    element.onclick = () => selectPodcast(Number(element.dataset.id));
  });
  document.querySelectorAll("[data-purge-podcast]").forEach((button) => {
    button.onclick = async (event) => {
      event.stopPropagation();
      if (await confirmAction("Delete downloads for this podcast?")) await App.PurgeCache("podcast", Number(button.dataset.purgePodcast));
    };
  });
  document.querySelectorAll("[data-unsubscribe]").forEach((button) => {
    button.onclick = async (event) => {
      event.stopPropagation();
      const podcastID = Number(button.dataset.unsubscribe);
      if (!(await confirmAction("Unsubscribe and remove all downloaded episodes for this podcast?"))) return;
      try {
        await App.Unsubscribe(podcastID);
        if (state.podcast?.id === podcastID) {
          state.podcast = null;
          state.episodes = [];
          $("#active-podcast").replaceChildren();
          $("#episodes").innerHTML = `<p class="muted">Select a podcast to browse episodes.</p>`;
        }
        await loadPodcasts();
      } catch (error) {
        showError(error);
      }
    };
  });
  if (!state.podcast && podcasts.length) await selectPodcast(podcasts[0].id);
  return podcasts;
}

async function selectPodcast(id) {
  const podcasts = (await App.ListPodcasts()) || [];
  state.podcast = podcasts.find((podcast) => podcast.id === id);
  if (!state.podcast) {
    state.episodes = [];
    $("#active-podcast").replaceChildren();
    $("#episodes").innerHTML = `<p class="muted">Select a podcast to browse episodes.</p>`;
    return;
  }
  state.episodes = (await App.ListEpisodes(id)) || [];
  $("#active-podcast").innerHTML = `<div class="active-podcast-header">${state.podcast.artworkUrl ? `<img src="${escapeHTML(state.podcast.artworkUrl)}" alt="" class="artwork">` : `<div class="artwork artwork-placeholder">P</div>`}<div><p class="eyebrow">Now browsing</p><h2>${escapeHTML(state.podcast.title)}</h2><p class="muted">${escapeHTML(state.podcast.description)}</p></div></div>`;
  $("#episodes").innerHTML = state.episodes.length ? state.episodes.map((episode) =>
    `<article class="episode"><div class="episode-main"><div class="episode-heading"><h3>${episode.isFavorite ? "★ " : ""}${escapeHTML(episode.title)}</h3><div class="episode-actions"><button class="pill-button" data-play="${episode.id}">Play</button><button class="pill-button" data-favorite="${episode.id}">${episode.isFavorite ? "Unstar" : "Star"}</button><button class="pill-button" data-purge-episode="${episode.id}">Purge</button></div></div><p>${escapeHTML(episode.description)}</p><textarea data-note="${episode.id}" placeholder="Add a note">${escapeHTML(episode.notes)}</textarea><button class="text-button" data-save-note="${episode.id}">Save note</button></div></article>`
  ).join("") : `<p class="muted">No episodes found in this feed.</p>`;
  document.querySelectorAll("[data-play]").forEach((button) => {
    button.onclick = () => {
      const episode = state.episodes.find((item) => item.id === Number(button.dataset.play));
      if (episode) return play(episode.id, Math.max(0, episode.positionSeconds || 0));
    };
  });
  document.querySelectorAll("[data-favorite]").forEach((button) => {
    button.onclick = async () => { await App.ToggleFavorite(Number(button.dataset.favorite)); await selectPodcast(id); };
  });
  document.querySelectorAll("[data-save-note]").forEach((button) => {
    button.onclick = async () => {
      const episodeID = Number(button.dataset.saveNote);
      await App.SaveNote(episodeID, document.querySelector(`[data-note="${episodeID}"]`).value);
      await selectPodcast(id);
    };
  });
  document.querySelectorAll("[data-purge-episode]").forEach((button) => {
    button.onclick = async () => { if (await confirmAction("Delete this downloaded episode?")) { await App.PurgeCache("episode", Number(button.dataset.purgeEpisode)); await selectPodcast(id); } };
  });
  await loadPodcasts();
}

async function play(id, position = 0) {
  const episode = state.episodes.find((item) => item.id === id);
  if (!episode) return;
  try {
    const source = await App.GetAudioSource(id);
    state.episode = episode;
    $("#audio").src = source;
    $("#now-playing").textContent = episode.title;
    const setPosition = () => {
      const audio = $("#audio");
      if (position > 0 && Number.isFinite(audio.duration)) {
        audio.currentTime = Math.min(position, audio.duration);
      } else if (position > 0) {
        audio.currentTime = position;
      }
    };
    $("#audio").addEventListener("loadedmetadata", setPosition, { once: true });
    setPosition();
    await $("#audio").play().catch((error) => {
      console.warn("Audio autoplay was not permitted; playback is ready to start.", error);
    });
  } catch (error) {
    state.episode = null;
    $("#audio").removeAttribute("src");
    $("#audio").load();
    $("#now-playing").textContent = "Nothing playing";
    throw error;
  }
}

$("#subscribe").onsubmit = async (event) => {
  event.preventDefault();
  try {
    await App.Subscribe($("#feed-url").value);
    $("#feed-url").value = "";
    await loadPodcasts();
  } catch (error) {
    showError(error);
  }
};

$("#refresh-all").onclick = async () => {
  await App.RefreshAllFeeds();
  await loadPodcasts();
  if (state.podcast) await selectPodcast(state.podcast.id);
};
$("#purge-all").onclick = async () => { if (await confirmAction("Delete all downloaded media?")) await App.PurgeCache("all", 0); };
$("#itunes-search").onsubmit = async (event) => {
  event.preventDefault();
  try {
    const results = await App.SearchITunes($("#search-query").value);
    $("#search-view").hidden = false;
    $("#podcasts").hidden = true;
    $("#search-results").innerHTML = results.length ? results.map((result) =>
      `<div class="search-result"><strong>${escapeHTML(result.title)}</strong><button data-search-feed="${escapeHTML(result.feedUrl)}">Subscribe</button></div>`
    ).join("") : `<p class="muted">No podcasts found.</p>`;
    document.querySelectorAll("[data-search-feed]").forEach((button) => {
      button.onclick = async () => { await App.Subscribe(button.dataset.searchFeed); $("#search-results").replaceChildren(); await loadPodcasts(); };
    });
  } catch (error) { showError(error); }
};
$("#close-search").onclick = () => { $("#search-view").hidden = true; $("#podcasts").hidden = false; $("#search-results").replaceChildren(); $("#search-query").value = ""; };
$("#import-opml").onclick = async () => { try { await App.ImportOPMLDialog(); await loadPodcasts(); } catch (error) { showError(error); } };
$("#export-opml").onclick = async () => { try { await App.ExportOPMLDialog(); await App.ShowInfo("OPML subscriptions exported."); } catch (error) { showError(error); } };

$("#audio").addEventListener("timeupdate", () => {
  const episode = state.episode;
  if (episode && Math.floor($("#audio").currentTime) % 5 === 0) {
    App.SaveProgress(episode.id, Math.floor($("#audio").currentTime), false);
  }
});
$("#audio").addEventListener("ended", () => {
  const episode = state.episode;
  if (episode) App.SaveProgress(episode.id, Math.floor($("#audio").duration), true);
});

async function restoreLastPlayed() {
  try {
    const lastPlayed = await App.GetLastPlayed();
    if (!lastPlayed?.id) return;
    await selectPodcast(lastPlayed.podcastId);
    await play(lastPlayed.id, Math.max(0, lastPlayed.positionSeconds || 0));
  } catch (error) {
    console.warn("Unable to restore last played episode; using default player state.", error);
    state.episode = null;
    $("#audio").removeAttribute("src");
    $("#audio").load();
    $("#now-playing").textContent = "Nothing playing";
  }
}

loadPodcasts().then(restoreLastPlayed).catch((error) => {
  $("#episodes").innerHTML = `<p class="muted">${escapeHTML(messageText(error))}</p>`;
  showError(error);
});
