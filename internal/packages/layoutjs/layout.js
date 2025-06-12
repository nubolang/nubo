document.addEventListener("DOMContentLoaded", () => {
  const loadPartial = (targetUrl, push = true) => {
    const url = new URL(targetUrl, location.origin);
    url.searchParams.set("__nubo_fragment", "partial");

    fetch(url.toString())
      .then((res) => res.text())
      .then((text) => {
        document.body.innerHTML = text;
        if (push) history.pushState({}, "", targetUrl);
        setupNTags();
      })
      .catch(() => {
        window.location.href = targetUrl;
      });
  };

  const setupNTags = () => {
    document.querySelectorAll("a[nubo-link]").forEach((link) => {
      link.addEventListener("click", (e) => {
        e.preventDefault();
        loadPartial(link.href);
      });
    });
  };

  window.addEventListener("popstate", () => {
    loadPartial(location.href, false);
  });

  setupNTags();
});
