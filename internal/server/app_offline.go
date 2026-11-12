package server

import "net/http"

// Sameway on a phone's home screen lives on a computer: asleep, off, or
// out of reach when the phone has left the Wi-Fi. The app opened on the
// browser's own error then, which reads as Sameway broken. Where a browser
// lets a page keep a worker (https, as through Tailscale, or this
// computer), it keeps one that answers a page that cannot be reached with
// a page saying why and Try again. It keeps nothing of the workspace:
// every page still comes from the computer, never from a copy, so nothing
// is stale and nothing private is left on the phone.

const offlineScript = `"use strict";
const page = '<!doctype html><html lang="en-GB"><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="color-scheme" content="light dark"><title>Sameway cannot be reached</title>' +
  '<main style="font:1.1rem/1.5 system-ui;max-width:30rem;margin:3rem auto;padding:0 1rem"><h1 style="font-size:1.5rem">Sameway cannot be reached just now</h1>' +
  // On the computer itself it is Sameway that has stopped, not the way to it.
  (/^(localhost|127\.0\.0\.1|\[::1\])$/.test(self.location.hostname)
    ? '<p>Sameway is not running on this computer. Open Sameway again, the way you opened it before, then press Try again.</p>'
    : '<p>Sameway lives on your computer. It answers when the computer is on and awake, and this phone can reach it: on the same Wi-Fi, or through Tailscale.</p>') +
  '<p><button onclick="location.reload()" style="font:inherit;min-height:44px;padding:0 1.25rem">Try again</button></p></main>';
self.addEventListener("install", () => self.skipWaiting());
self.addEventListener("activate", (e) => e.waitUntil(self.clients.claim()));
self.addEventListener("fetch", (e) => {
  if (e.request.mode !== "navigate" || e.request.method !== "GET") return; // a form sent, an upload: left to the browser
  e.respondWith(fetch(e.request).catch(() => new Response(page, { status: 503, headers: { "Content-Type": "text/html; charset=utf-8" } })));
});
`

// offlineWorker is the worker, served from the top so it covers every page.
func offlineWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(offlineScript))
}
