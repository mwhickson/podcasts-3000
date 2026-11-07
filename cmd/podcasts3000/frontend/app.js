import * as App from "./wailsjs/go/podcasts/App.js";

const state = { podcast: null, episodes: [], episode: null, filters: { search: "", showAll: false, sort: "pub-desc" } };
const sidebarWidth = { min: 220, max: 560, default: 310 };
const $ = (selector) => document.querySelector(selector);
const escapeHTML = (value) => {
  const element = document.createElement("div");
  element.textContent = value || "";
  return element.innerHTML;
};
const messageText = (error) => error instanceof Error ? error.message : String(error);
const setStatus = (message, type = "") => {
  const status = $("#status-bar");
  status.textContent = message;
  status.className = `status-bar${type ? ` ${type}` : ""}`;
};
const showError = (error) => {
  console.error(error);
  setStatus(`Error: ${messageText(error)}`, "error");
};
const confirmAction = (message) => App.Confirm(message).catch((error) => {
  console.error(error);
  showError(error);
  return false;
});
const resetEpisodeFilters = () => {
  state.filters = { search: "", showAll: false, sort: "pub-desc" };
  $("#episode-search").value = "";
  $("#unplayed-only").checked = false;
  $("#episode-sort").value = "pub-desc";
};
const filteredEpisodes = () => {
  const query = state.filters.search.trim().toLocaleLowerCase();
  return state.episodes
    .map((episode) => {
      const titleMatch = query !== "" && episode.title.toLocaleLowerCase().includes(query);
      const descriptionMatch = query !== "" && episode.description.toLocaleLowerCase().includes(query);
      return { episode, matchRank: query === "" ? 0 : titleMatch ? 0 : descriptionMatch ? 1 : -1 };
    })
    .filter(({ episode, matchRank }) => matchRank >= 0 && (state.filters.showAll || !episode.isCompleted))
    .sort((left, right) => {
      if (left.matchRank !== right.matchRank) return left.matchRank - right.matchRank;
      if (state.filters.sort.startsWith("name")) {
        const direction = state.filters.sort === "name-asc" ? 1 : -1;
        return left.episode.title.localeCompare(right.episode.title) * direction;
      }
      const leftDate = left.episode.pubDate ? Date.parse(left.episode.pubDate) || 0 : 0;
      const rightDate = right.episode.pubDate ? Date.parse(right.episode.pubDate) || 0 : 0;
      const direction = state.filters.sort === "pub-asc" ? 1 : -1;
      return (leftDate - rightDate) * direction;
    })
    .map(({ episode }) => episode);
};
const formatDate = (value) => value ? new Date(value).toLocaleDateString() : "Publication date unknown";
const formatDuration = (seconds) => {
  const total = Math.max(0, Number(seconds) || 0);
  return total ? `${Math.floor(total / 60)}m ${String(total % 60).padStart(2, "0")}s` : "Duration unknown";
};
const renderEpisodes = () => {
  const episodes = filteredEpisodes();
  $("#episodes").innerHTML = episodes.length ? episodes.map((episode) =>
    `<details class="episode ${episode.isCompleted ? "played" : ""} ${state.episode?.id === episode.id ? "playing" : ""}" data-episode="${episode.id}"><summary class="episode-heading"><h3>${episode.isFavorite ? "★ " : ""}${escapeHTML(episode.title)} <span class="episode-meta">${formatDate(episode.pubDate)}</span></h3><div class="episode-actions"><button class="pill-button" data-play="${episode.id}">Play</button><button class="pill-button" data-played="${episode.id}">${episode.isCompleted ? "Mark unplayed" : "Mark played"}</button><button class="pill-button" data-favorite="${episode.id}">${episode.isFavorite ? "Unstar" : "Star"}</button>${episode.isDownloaded ? `<button class="pill-button" data-purge-episode="${episode.id}">Purge</button>` : ""}</div></summary><div class="episode-body"><p class="episode-meta">${formatDuration(episode.durationSeconds)}</p><p>${escapeHTML(episode.description)}</p><textarea data-note="${episode.id}" placeholder="Add a note">${escapeHTML(episode.notes)}</textarea><button class="text-button" data-save-note="${episode.id}">Save note</button></div></details>`
  ).join("") : `<p class="muted">${state.episodes.length ? "No episodes match the current filters." : "No episodes found in this feed."}</p>`;
};
const updatePlayingPodcast = () => {
  document.querySelectorAll(".podcast").forEach((element) => {
    const podcastID = Number(element.dataset.id);
    element.classList.toggle("playing", state.episode?.podcastId === podcastID);
  });
};
const setSidebarWidth = (width) => {
  const clamped = Math.min(sidebarWidth.max, Math.max(sidebarWidth.min, Math.round(width)));
  document.documentElement.style.setProperty("--sidebar-width", `${clamped}px`);
  $("#pane-resizer").setAttribute("aria-valuenow", String(clamped));
  localStorage.setItem("podcasts3000.sidebar-width", String(clamped));
};
const resizeLayout = () => {
  document.documentElement.style.setProperty("--viewport-height", `${window.innerHeight}px`);
  if (state.podcast) {
    renderEpisodes();
    bindEpisodeActions(state.podcast.id);
  }
};
const initializeLayout = () => {
  const savedWidth = Number(localStorage.getItem("podcasts3000.sidebar-width"));
  setSidebarWidth(Number.isFinite(savedWidth) && savedWidth > 0 ? savedWidth : sidebarWidth.default);
  resizeLayout();
  const resizer = $("#pane-resizer");
  let dragging = false;
  const stopDragging = () => {
    dragging = false;
    document.body.classList.remove("resizing-panes");
  };
  resizer.addEventListener("pointerdown", (event) => {
    dragging = true;
    resizer.setPointerCapture(event.pointerId);
    document.body.classList.add("resizing-panes");
    event.preventDefault();
  });
  resizer.addEventListener("pointermove", (event) => {
    if (dragging) setSidebarWidth(event.clientX);
  });
  resizer.addEventListener("pointerup", stopDragging);
  resizer.addEventListener("pointercancel", stopDragging);
  resizer.addEventListener("keydown", (event) => {
    if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
      setSidebarWidth(Number(getComputedStyle(document.documentElement).getPropertyValue("--sidebar-width").replace("px", "")) + (event.key === "ArrowLeft" ? -10 : 10));
      event.preventDefault();
    }
  });
  window.addEventListener("resize", resizeLayout);
};
const bindEpisodeActions = (podcastID) => {
  document.querySelectorAll(".episode-actions button").forEach((button) => {
    button.onclick = (event) => event.stopPropagation();
  });
  document.querySelectorAll("[data-play]").forEach((button) => {
    button.onclick = (event) => {
      event.stopPropagation();
      const episode = state.episodes.find((item) => item.id === Number(button.dataset.play));
      if (episode) return play(episode.id, Math.max(0, episode.positionSeconds || 0));
    };
  });
  document.querySelectorAll("[data-favorite]").forEach((button) => {
    button.onclick = async (event) => {
      event.stopPropagation();
      try {
        await App.ToggleFavorite(Number(button.dataset.favorite));
        setStatus("Favorite updated", "success");
        await selectPodcast(podcastID);
      } catch (error) { showError(error); }
    };
  });
  document.querySelectorAll("[data-played]").forEach((button) => {
    button.onclick = async (event) => {
      event.stopPropagation();
      const episode = state.episodes.find((item) => item.id === Number(button.dataset.played));
      if (!episode) return;
      try {
        await App.SaveProgress(episode.id, episode.isCompleted ? 0 : Math.max(0, episode.durationSeconds || 0), !episode.isCompleted);
        episode.isCompleted = !episode.isCompleted;
        renderEpisodes();
        bindEpisodeActions(podcastID);
        setStatus(episode.isCompleted ? "Episode marked played" : "Episode marked unplayed", "success");
      } catch (error) { showError(error); }
    };
  });
  document.querySelectorAll("[data-save-note]").forEach((button) => {
    button.onclick = async (event) => {
      event.stopPropagation();
      const episodeID = Number(button.dataset.saveNote);
      try {
        await App.SaveNote(episodeID, document.querySelector(`[data-note="${episodeID}"]`).value);
        setStatus("Note saved", "success");
        await selectPodcast(podcastID);
      } catch (error) { showError(error); }
    };
  });
  document.querySelectorAll("[data-purge-episode]").forEach((button) => {
    button.onclick = async (event) => {
      event.stopPropagation();
      if (!(await confirmAction("Delete this downloaded episode?"))) return;
      try {
        setStatus("Purging episode download…");
        await App.PurgeCache("episode", Number(button.dataset.purgeEpisode));
        setStatus("Episode download purged", "success");
        await selectPodcast(podcastID);
      } catch (error) { showError(error); }
    };
  });
};

async function loadPodcasts() {
  setStatus("Loading subscriptions…");
  const podcasts = (await App.ListPodcasts()) || [];
  $("#podcasts").innerHTML = podcasts.map((podcast) =>
    `<div class="podcast ${state.podcast?.id === podcast.id ? "active" : ""} ${state.episode?.podcastId === podcast.id ? "playing" : ""} ${podcast.hasRecentUnplayed ? "recent-unplayed" : ""}" data-id="${podcast.id}"><span class="podcast-name">${escapeHTML(podcast.title)}</span><span class="podcast-actions"><button class="compact-button" data-mark-podcast-played="${podcast.id}" title="Mark all episodes played">Mark Played</button>${podcast.hasCachedEpisodes ? `<button class="compact-button" data-purge-podcast="${podcast.id}" title="Purge this podcast's downloads">Purge</button>` : ""}<button class="compact-button" data-unsubscribe="${podcast.id}" title="Unsubscribe from this podcast">Unsubscribe</button></span></div>`
  ).join("");
  document.querySelectorAll(".podcast").forEach((element) => {
    element.onclick = () => selectPodcast(Number(element.dataset.id));
  });
  document.querySelectorAll("[data-purge-podcast]").forEach((button) => {
    button.onclick = async (event) => {
      event.stopPropagation();
      if (!(await confirmAction("Delete downloads for this podcast?"))) return;
      try {
        setStatus("Purging podcast downloads…");
        await App.PurgeCache("podcast", Number(button.dataset.purgePodcast));
        await loadPodcasts();
        if (state.podcast?.id === Number(button.dataset.purgePodcast)) await selectPodcast(state.podcast.id);
        setStatus("Podcast downloads purged", "success");
      } catch (error) { showError(error); }
    };
  });
  document.querySelectorAll("[data-mark-podcast-played]").forEach((button) => {
    button.onclick = async (event) => {
      event.stopPropagation();
      try {
        await App.MarkPodcastPlayed(Number(button.dataset.markPodcastPlayed));
        await loadPodcasts();
        if (state.podcast?.id === Number(button.dataset.markPodcastPlayed)) await selectPodcast(state.podcast.id);
        setStatus("All podcast episodes marked played", "success");
      } catch (error) { showError(error); }
    };
  });
  document.querySelectorAll("[data-unsubscribe]").forEach((button) => {
    button.onclick = async (event) => {
      event.stopPropagation();
      const podcastID = Number(button.dataset.unsubscribe);
      if (!(await confirmAction("Unsubscribe and remove all downloaded episodes for this podcast?"))) return;
      try {
        setStatus("Unsubscribing…");
        await App.Unsubscribe(podcastID);
        if (state.podcast?.id === podcastID) {
          state.podcast = null;
          state.episodes = [];
          state.episode = null;
          $("#episode-toolbar").hidden = true;
          $("#active-podcast").replaceChildren();
          $("#episodes").innerHTML = `<p class="muted">Select a podcast to browse episodes.</p>`;
        }
        await loadPodcasts();
        setStatus("Podcast unsubscribed", "success");
      } catch (error) {
        showError(error);
      }
    };
  });
  if (!state.podcast && podcasts.length) await selectPodcast(podcasts[0].id);
  setStatus("Subscriptions loaded", "success");
  return podcasts;
}

async function selectPodcast(id) {
  resetEpisodeFilters();
  setStatus("Loading episodes…");
  const podcasts = (await App.ListPodcasts()) || [];
  state.podcast = podcasts.find((podcast) => podcast.id === id);
  if (!state.podcast) {
    state.episodes = [];
    state.episode = null;
    $("#episode-toolbar").hidden = true;
    $("#active-podcast").replaceChildren();
    $("#episodes").innerHTML = `<p class="muted">Select a podcast to browse episodes.</p>`;
    return;
  }
  state.episodes = (await App.ListEpisodes(id)) || [];
  setStatus(`${state.episodes.length} episode${state.episodes.length === 1 ? "" : "s"} loaded`, "success");
  $("#episode-toolbar").hidden = false;
  $("#active-podcast").innerHTML = `<div class="active-podcast-header">${state.podcast.artworkUrl ? `<img src="${escapeHTML(state.podcast.artworkUrl)}" alt="" class="artwork">` : `<div class="artwork artwork-placeholder">P</div>`}<div><p class="eyebrow">Now browsing</p><h2>${escapeHTML(state.podcast.title)}</h2><p class="muted">${escapeHTML(state.podcast.description)}</p></div></div>`;
  renderEpisodes();
  bindEpisodeActions(id);
  await loadPodcasts();
}

$("#episode-search").addEventListener("input", (event) => {
  state.filters.search = event.target.value;
  renderEpisodes();
  bindEpisodeActions(state.podcast?.id);
});
$("#unplayed-only").addEventListener("change", (event) => {
  state.filters.showAll = event.target.checked;
  renderEpisodes();
  bindEpisodeActions(state.podcast?.id);
});
$("#episode-sort").addEventListener("change", (event) => {
  state.filters.sort = event.target.value;
  renderEpisodes();
  bindEpisodeActions(state.podcast?.id);
});

async function play(id, position = 0, autoStart = true) {
  const episode = state.episodes.find((item) => item.id === id);
  if (!episode) return;
  try {
    setStatus("Buffering episode…");
    const source = await App.GetAudioSource(id);
    state.episode = episode;
    renderEpisodes();
    bindEpisodeActions(episode.podcastId);
    updatePlayingPodcast();
    $("#audio").src = source;
    $("#now-playing").disabled = false;
    $("#now-playing").textContent = `${state.podcast?.title || "Podcast"} — ${episode.title}`;
    const setPosition = () => {
      const audio = $("#audio");
      if (Number.isFinite(audio.duration) && audio.duration > 0) {
        const duration = Math.floor(audio.duration);
        episode.durationSeconds = duration;
        App.UpdateDuration(episode.id, duration).catch(showError);
      }
      if (position > 0 && Number.isFinite(audio.duration)) {
        audio.currentTime = Math.min(position, audio.duration);
      } else if (position > 0) {
        audio.currentTime = position;
      }
    };
    $("#audio").addEventListener("loadedmetadata", setPosition, { once: true });
    setPosition();
    if (autoStart) {
      await $("#audio").play().catch((error) => {
        console.warn("Audio autoplay was not permitted; playback is ready to start.", error);
      });
      setStatus(`Playing: ${episode.title}`, "success");
    } else {
      setStatus(`Ready to play: ${episode.title}`, "success");
    }
  } catch (error) {
    state.episode = null;
    $("#audio").removeAttribute("src");
    $("#audio").load();
    $("#now-playing").disabled = true;
    $("#now-playing").textContent = "Nothing playing";
    throw error;
  }
}

$("#now-playing").onclick = async () => {
  if (!state.episode) return;
  try {
    await selectPodcast(state.episode.podcastId);
    const episodeElement = document.querySelector(`[data-episode="${state.episode.id}"]`);
    if (episodeElement) {
      episodeElement.open = true;
      episodeElement.scrollIntoView({ behavior: "smooth", block: "nearest" });
    }
  } catch (error) { showError(error); }
};

$("#subscribe").onsubmit = async (event) => {
  event.preventDefault();
  try {
    setStatus("Subscribing and retrieving episodes…");
    await App.Subscribe($("#feed-url").value);
    $("#feed-url").value = "";
    await loadPodcasts();
    setStatus("Podcast subscribed", "success");
  } catch (error) {
    showError(error);
  }
};

$("#refresh-all").onclick = async () => {
  try {
    setStatus("Refreshing subscriptions…");
    await App.RefreshAllFeeds();
    await loadPodcasts();
    if (state.podcast) await selectPodcast(state.podcast.id);
    setStatus("Subscriptions refreshed", "success");
  } catch (error) { showError(error); }
};
$("#purge-all").onclick = async () => {
  if (!(await confirmAction("Delete all downloaded media?"))) return;
  try {
    setStatus("Purging all downloads…");
    await App.PurgeCache("all", 0);
    setStatus("All downloads purged", "success");
  } catch (error) { showError(error); }
};
$("#itunes-search").onsubmit = async (event) => {
  event.preventDefault();
  try {
    setStatus("Searching iTunes…");
    const results = await App.SearchITunes($("#search-query").value);
    $("#search-view").hidden = false;
    $("#podcasts").hidden = true;
    $("#search-results").innerHTML = results.length ? results.map((result) =>
      `<div class="search-result"><strong>${escapeHTML(result.title)}</strong><button data-search-feed="${escapeHTML(result.feedUrl)}">Subscribe</button></div>`
    ).join("") : `<p class="muted">No podcasts found.</p>`;
    document.querySelectorAll("[data-search-feed]").forEach((button) => {
      button.onclick = async () => {
        try {
          setStatus("Subscribing and retrieving episodes…");
          await App.Subscribe(button.dataset.searchFeed);
          $("#search-results").replaceChildren();
          await loadPodcasts();
          setStatus("Podcast subscribed", "success");
        } catch (error) { showError(error); }
      };
    });
    setStatus(`${results.length} search result${results.length === 1 ? "" : "s"} found`, "success");
  } catch (error) { showError(error); }
};
$("#close-search").onclick = () => { $("#search-view").hidden = true; $("#podcasts").hidden = false; $("#search-results").replaceChildren(); $("#search-query").value = ""; };
$("#import-opml").onclick = async () => {
  try {
    setStatus("Importing subscriptions…");
    const results = await App.ImportOPMLDialog();
    await loadPodcasts();
    const failures = (results || []).filter((result) => result.error);
    if (failures.length) {
      const details = failures.map((result) => `${result.feedUrl}: ${result.error}`).join(" | ");
      setStatus(`Imported ${results.length - failures.length} subscription${results.length - failures.length === 1 ? "" : "s"}; ${failures.length} failed: ${details}`, "error");
    } else {
      setStatus(`Imported ${results?.length || 0} subscription${results?.length === 1 ? "" : "s"}`, "success");
    }
  } catch (error) { showError(error); }
};
$("#export-opml").onclick = async () => { try { setStatus("Exporting subscriptions…"); await App.ExportOPMLDialog(); setStatus("Subscriptions exported", "success"); } catch (error) { showError(error); } };

$("#audio").addEventListener("timeupdate", () => {
  const episode = state.episode;
  if (episode && Math.floor($("#audio").currentTime) % 5 === 0) {
    App.SaveProgress(episode.id, Math.floor($("#audio").currentTime), false);
  }
});
$("#audio").addEventListener("ended", async () => {
  const episode = state.episode;
  if (!episode) return;
  try {
    await App.SaveProgress(episode.id, Math.floor($("#audio").duration), true);
    episode.isCompleted = true;
    episode.isDownloaded = false;
    renderEpisodes();
    bindEpisodeActions(episode.podcastId);
    await loadPodcasts();
    setStatus("Episode completed and download purged", "success");
  } catch (error) { showError(error); }
});

async function restoreLastPlayed() {
  try {
    const lastPlayed = await App.GetLastPlayed();
    if (!lastPlayed?.id) return;
    await selectPodcast(lastPlayed.podcastId);
    await play(lastPlayed.id, Math.max(0, lastPlayed.positionSeconds || 0), false);
  } catch (error) {
    console.warn("Unable to restore last played episode; using default player state.", error);
    state.episode = null;
    $("#audio").removeAttribute("src");
    $("#audio").load();
    $("#now-playing").disabled = true;
    $("#now-playing").textContent = "Nothing playing";
  }
}

resetEpisodeFilters();
initializeLayout();
loadPodcasts().then(restoreLastPlayed).catch((error) => {
  $("#episodes").innerHTML = `<p class="muted">${escapeHTML(messageText(error))}</p>`;
  showError(error);
});
