// Drafts.
//
// Words typed and not yet sent or saved survive leaving the page: what is
// in the chat composer, and an inline edit in progress. They are kept in
// this browser, under the page's origin, keyed by what they belong to (a
// chat, a block or a record), and go the moment they are sent or saved,
// or when the person throws them away. Nothing here is sent anywhere; a
// draft is the person's until they decide.
(function () {
  "use strict";
  var PREFIX = "sameway:draft:";
  function get(k) { try { return JSON.parse(localStorage.getItem(PREFIX + k)); } catch (e) { return null; } }
  function set(k, v) { try { localStorage.setItem(PREFIX + k, JSON.stringify(v)); } catch (e) { /* a private window keeps nothing */ } }
  function del(k) { try { localStorage.removeItem(PREFIX + k); } catch (e) { /* fine */ } }
  function el(html) {
    var t = document.createElement("template");
    t.innerHTML = html.trim();
    return t.content.firstElementChild;
  }

  // The composer: one draft per chat, restored when the box is empty and
  // cleared when the message is sent.
  function compose(form) {
    var ta = form.querySelector("textarea");
    if (!ta || form._drafted) return;
    form._drafted = true;
    var k = "chat:" + (form.getAttribute("data-chat") || "current");
    var saved = get(k);
    if (typeof saved === "string" && saved && !ta.value) ta.value = saved;
    ta.addEventListener("input", function () { if (ta.value.trim()) set(k, ta.value); else del(k); });
    form.addEventListener("submit", function () { del(k); });
  }

  // An inline edit: the fields of the form 08-edit.js builds, saved as
  // they change, put back when the same edit is opened again, and dropped
  // on Cancel, or once the page says the save went through. Pressing Save
  // is not enough: a save can be refused, and the words must be there to
  // fix.
  function editKey(block) {
    return "edit:" + (block.getAttribute("data-edit-action") || "/canvas/" + block.getAttribute("data-block-id") + "/props");
  }
  function fields(form) {
    var out = {};
    var data = new FormData(form);
    data.forEach(function (v, name) { if (typeof v === "string" && /^(prop|html|level)-/.test(name)) out[name] = v; });
    return out;
  }
  function watch(form) {
    if (form._drafted) return;
    form._drafted = true;
    var block = form.closest("[data-block-id]");
    if (!block) return;
    var k = editKey(block);
    var saved = get(k);
    if (saved && typeof saved === "object") {
      Object.keys(saved).forEach(function (name) {
        var f = form.elements[name];
        if (f && "value" in f && f.value !== saved[name]) f.value = saved[name];
        // Rich text is what the person sees and typed into, not the
        // hidden field that carries it: put the words back there too.
        var prose = f && f.closest && f.closest(".sw-prose-field");
        var editor = prose && /^html-/.test(name) && prose.querySelector(".sw-prose-editor");
        if (editor && saved[name]) editor.innerHTML = saved[name];
      });
    }
    var notice = block.querySelector(".sw-draft");
    if (notice) notice.remove();
    form.addEventListener("input", function () { set(k, fields(form)); });
    form.addEventListener("submit", function () { set(k, fields(form)); });
    form.addEventListener("keydown", function (e) { if (e.key === "Escape") del(k); });
    var cancel = form.querySelector("[data-cancel]");
    if (cancel) cancel.addEventListener("click", function () { del(k); });
  }

  // A block with an edit left unfinished says so, and offers to go on
  // with it or to let it go.
  function offer(block) {
    var k = editKey(block);
    var saved = get(k);
    if (!saved || block.querySelector(".sw-draft") || block.querySelector(".sw-inline-form")) return;
    var edit = block.querySelector("[data-edit]");
    if (!edit) return;
    var notice = el('<p class="sw-draft sw-small"><span class="sw-draft__word">You were editing this.</span> ' +
      '<button type="button" class="sw-button sw-button--quiet sw-pressable" data-draft-continue>Edit</button> ' +
      '<button type="button" class="sw-button sw-button--quiet sw-pressable" data-draft-discard>Discard</button></p>');
    notice.querySelector("[data-draft-continue]").addEventListener("click", function () { notice.remove(); edit.click(); });
    notice.querySelector("[data-draft-discard]").addEventListener("click", function () { del(k); notice.remove(); });
    block.insertBefore(notice, block.firstChild);
  }

  // What the person's last action came to (outcome.go): an edit saved
  // lets its draft go; an edit refused opens again, holding what they
  // typed, under the message that says why.
  function settle() {
    document.querySelectorAll(".sw-outcome[data-outcome-for]").forEach(function (o) {
      var k = "edit:" + o.getAttribute("data-outcome-for");
      if (o.getAttribute("data-outcome") === "done") { del(k); return; }
      if (!get(k)) return;
      var blocks = document.querySelectorAll("[data-block-id]");
      for (var i = 0; i < blocks.length; i++) {
        if (editKey(blocks[i]) !== k) continue;
        var edit = blocks[i].querySelector("[data-edit]");
        // The field the editor opens on is told why the edit was refused,
        // so a screen reader reads the reason with it.
        if (edit) setTimeout(function () {
          edit.click();
          var field = document.activeElement;
          if (field && field.closest(".sw-inline-form")) field.setAttribute("aria-describedby", "outcome");
        }, 0);
        return;
      }
    });
  }

  function init() {
    settle();
    document.querySelectorAll("form.sw-compose").forEach(compose);
    document.querySelectorAll("[data-block-id]").forEach(offer);
    document.querySelectorAll("form.sw-inline-form").forEach(watch);
    if (!window.MutationObserver) return;
    new MutationObserver(function (records) {
      records.forEach(function (r) {
        Array.prototype.forEach.call(r.addedNodes, function (n) {
          if (n.nodeType !== 1) return;
          if (n.matches && n.matches("form.sw-inline-form")) watch(n);
          if (n.querySelectorAll) n.querySelectorAll("form.sw-inline-form").forEach(watch);
        });
      });
    }).observe(document.body, { childList: true, subtree: true });
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
  document.addEventListener("sw:refresh", function () {
    document.querySelectorAll("form.sw-compose").forEach(compose);
    document.querySelectorAll("[data-block-id]").forEach(offer);
  });
})();
