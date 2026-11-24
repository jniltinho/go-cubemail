// Applies the theme this browser remembers before the first paint, so a reload in a dark theme shows no light
// frame. A separate file on purpose: the Content Security Policy is script-src 'self', which forbids inline scripts.
// The key and its shape are written by src/utils/theme.ts.
try {
  var cached = JSON.parse(localStorage.getItem('gorc_theme') || 'null')
  if (cached && /^[a-z0-9-]+$/.test(cached.id)) {
    var root = document.documentElement
    root.dataset.theme = cached.id
    if (cached.scheme === 'dark' || cached.scheme === 'light') {
      root.style.colorScheme = cached.scheme
      var meta = document.querySelector('meta[name="color-scheme"]')
      if (meta) meta.setAttribute('content', cached.scheme)
    }
  }
} catch {
  // storage blocked or the entry is damaged: the page starts with the default theme
}
