(function () {
  var modal = document.getElementById('modal');
  function close() {
    if (modal) modal.innerHTML = '';
    document.body.style.overflow = '';
  }
  document.addEventListener('click', function (e) {
    var t = e.target;
    if (!(t instanceof Element)) return;
    var closer = t.closest('[data-close]');
    if (closer && (closer.tagName === 'BUTTON' || t === closer)) close();
  });
  document.addEventListener('keydown', function (e) {
    if (!modal || !modal.firstElementChild) return;
    if (e.key === 'Escape') close();
    if (e.key === 'ArrowLeft') { var p = modal.querySelector('[data-prev]'); if (p) p.click(); }
    if (e.key === 'ArrowRight') { var n = modal.querySelector('[data-next]'); if (n) n.click(); }
  });
  document.body.addEventListener('htmx:afterSwap', function (e) {
    if (e.target === modal && modal.firstElementChild) {
      document.body.style.overflow = 'hidden';
      var b = modal.querySelector('[data-next]');
      if (b) b.focus({ preventScroll: true });
    }
  });
})();

(function () {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches || !('IntersectionObserver' in window)) return;
  var sel = '.sec-h, .case, .bento .tile, .gh-stats .tile, .heat-panel, .pgrp, .rev, .shot, .ct, .contact .mega';
  var vh = window.innerHeight;
  var io = new IntersectionObserver(function (entries) {
    entries.forEach(function (e) {
      if (!e.isIntersecting) return;
      e.target.classList.add('in');
      io.unobserve(e.target);
    });
  }, { threshold: 0.12, rootMargin: '0px 0px -6% 0px' });
  document.querySelectorAll(sel).forEach(function (el) {
    var r = el.getBoundingClientRect();
    if (r.top < vh * 0.9) return;
    var sib = el.parentElement ? Array.prototype.indexOf.call(el.parentElement.children, el) : 0;
    el.style.transitionDelay = Math.min(sib, 5) * 70 + 'ms';
    el.classList.add('rv');
    io.observe(el);
  });

  var cio = new IntersectionObserver(function (entries) {
    entries.forEach(function (e) {
      if (!e.isIntersecting) return;
      cio.unobserve(e.target);
      var el = e.target, final = el.textContent.trim();
      var target = parseInt(final.replace(/[^0-9]/g, ''), 10);
      if (!isFinite(target)) return;
      var comma = final.indexOf(',') !== -1, suffix = final.replace(/[0-9,]/g, '');
      var t0 = performance.now(), dur = 1000;
      function fmt(v) { return (comma ? v.toLocaleString('en-US') : String(v)) + suffix; }
      function step(t) {
        var p = Math.min(1, (t - t0) / dur), v = Math.round(target * (1 - Math.pow(1 - p, 3)));
        el.textContent = fmt(v);
        if (p < 1) requestAnimationFrame(step); else el.textContent = final;
      }
      el.textContent = fmt(0);
      requestAnimationFrame(step);
    });
  }, { threshold: 0.6 });
  document.querySelectorAll('[data-count]').forEach(function (el) { cio.observe(el); });
})();
