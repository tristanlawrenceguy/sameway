// What look puts in a page it reads with its scripts run: finding a
// control by the name a screen reader gives it, saying what has focus,
// and writing the page down as it stands. Run before every expression,
// so a page a step navigated to has it too.
(function () {
  "use strict";
  if (window.__look) return;
  function clean(s) { return (s || "").replace(/\s+/g, " ").trim(); }
  function shown(el) { return el.checkVisibility ? el.checkVisibility({ visibilityProperty: true }) : el.offsetParent !== null; }

  // name is the accessible name, the way look reads it from markup.
  function name(el) {
    var by = el.getAttribute("aria-labelledby");
    if (by) {
      var t = clean(by.split(/\s+/).map(function (id) { var n = document.getElementById(id); return n ? n.textContent : ""; }).join(" "));
      if (t) return t;
    }
    var label = clean(el.getAttribute("aria-label"));
    if (label) return label;
    if (el.labels && el.labels.length) return clean(Array.prototype.map.call(el.labels, function (l) { return l.textContent; }).join(" "));
    if (el.tagName === "INPUT" && /^(submit|button|reset)$/i.test(el.type)) return clean(el.value);
    var text = clean(el.innerText || el.textContent);
    if (text) return text;
    return clean(el.getAttribute("title") || el.getAttribute("placeholder") || el.getAttribute("alt"));
  }
  function kind(el) {
    var role = el.getAttribute("role");
    if (role) return role;
    var tag = el.tagName.toLowerCase();
    var kinds = { a: "link", button: "button", summary: "disclosure", select: "listbox", textarea: "textbox" };
    if (kinds[tag]) return kinds[tag];
    if (tag === "input") {
      var type = (el.type || "text").toLowerCase();
      if (/^(submit|button|reset|image)$/.test(type)) return "button";
      return type === "checkbox" || type === "radio" ? type : "textbox";
    }
    return el.isContentEditable ? "textbox" : tag;
  }
  function say(el) {
    if (!el || el === document.body || el === document.documentElement) return "";
    return kind(el) + ": " + name(el);
  }

  var OPERABLE = 'a[href],button,summary,input:not([type=hidden]),select,textarea,[role=button],[role=link],[role=tab],[role=menuitem],[role=checkbox],[contenteditable=""],[contenteditable=true],[tabindex]:not([tabindex="-1"])';
  var FIELDS = 'input:not([type=hidden]):not([type=submit]):not([type=button]),textarea,select,[contenteditable=""],[contenteditable=true],[role=textbox]';
  function find(wanted, fields) {
    var all = Array.prototype.filter.call(document.querySelectorAll(fields ? FIELDS : OPERABLE), shown);
    var w = clean(wanted).toLowerCase();
    // Exactly the name, or else the shortest name with it in: "Edit"
    // is the Edit button, not a region that has one.
    var el = all.filter(function (e) { return name(e).toLowerCase() === w; })[0] ||
      all.filter(function (e) { return name(e).toLowerCase().indexOf(w) >= 0; })
        .sort(function (a, b) { return name(a).length - name(b).length; })[0];
    if (el) return { el: el };
    return { error: "nothing " + (fields ? "to type into" : "to press") + ' is called "' + wanted + '"; the page has ' +
      (all.slice(0, 80).map(say).join("; ") || "nothing of the kind shown") };
  }

  var ids = 0;
  window.__look = {
    target: function (wanted, fields) {
      var f = find(wanted, fields);
      if (!f.el) return f;
      f.el.scrollIntoView({ block: "center" });
      var r = f.el.getBoundingClientRect();
      return { x: r.left + r.width / 2, y: r.top + r.height / 2, what: say(f.el) };
    },
    focus: function (wanted) {
      var f = find(wanted, true);
      if (!f.el) return f;
      f.el.focus();
      return { what: say(f.el) };
    },
    focused: function () { return say(document.activeElement); },
    // at is what has focus, with an id that stays with the element.
    at: function () {
      var el = document.activeElement;
      if (!el || el === document.body || el === document.documentElement) return { id: 0 };
      if (!el.__lookId) el.__lookId = ++ids;
      return { id: el.__lookId, says: say(el) };
    },
    // top puts the place Tab starts from at the top of the page. The
    // browser remembers where the last press or focus was, so a blur is
    // not enough: a mark at the start of the body takes focus, and goes
    // once Tab has left it.
    top: function () {
      if (window.getSelection) getSelection().removeAllRanges();
      window.scrollTo(0, 0);
      var mark = document.createElement("span");
      mark.tabIndex = -1;
      document.body.insertBefore(mark, document.body.firstChild);
      mark.focus({ preventScroll: true });
      mark.addEventListener("blur", function () { mark.remove(); });
    },
    // settled is a frame drawn and a moment after, for scripts that wait
    // a tick before they change the page.
    settled: function () {
      return new Promise(function (done) { requestAnimationFrame(function () { setTimeout(done, 350); }); });
    },
    // snapshot is the page as it stands, in markup: what is not shown is
    // marked where it stops being shown, and each field says its value.
    snapshot: function () {
      document.querySelectorAll("body *").forEach(function (el) {
        if (!shown(el) && (!el.parentElement || shown(el.parentElement))) el.setAttribute("data-look-unseen", "");
      });
      document.querySelectorAll("input").forEach(function (el) {
        if (el.type === "checkbox" || el.type === "radio") el.toggleAttribute("checked", el.checked);
        else if (el.type !== "password" && el.type !== "file") el.setAttribute("value", el.value);
      });
      document.querySelectorAll("textarea").forEach(function (el) { el.textContent = el.value; });
      document.querySelectorAll("select option").forEach(function (o) { o.toggleAttribute("selected", o.selected); });
      return "<!DOCTYPE html>\n" + document.documentElement.outerHTML;
    }
  };
})()
