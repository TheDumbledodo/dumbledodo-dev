(function () {
    const modal = document.getElementById('modal');

    function close() {
        if (modal) modal.innerHTML = '';
        document.body.style.overflow = '';
    }

    document.addEventListener('click', function (e) {
        const t = e.target;
        if (!(t instanceof Element)) return;
        const closer = t.closest('[data-close]');
        if (closer && (closer.tagName === 'BUTTON' || t === closer)) close();
    });
    document.addEventListener('keydown', function (e) {
        if (!modal || !modal.firstElementChild) return;
        if (e.key === 'Escape') close();
        if (e.key === 'ArrowLeft') {
            const p = modal.querySelector('[data-prev]');
            if (p) p.click();
        }
        if (e.key === 'ArrowRight') {
            const n = modal.querySelector('[data-next]');
            if (n) n.click();
        }
    });
    document.body.addEventListener('htmx:afterSwap', function (e) {
        if (e.target === modal && modal.firstElementChild) {
            document.body.style.overflow = 'hidden';
            const b = modal.querySelector('[data-next]');
            if (b) b.focus({preventScroll: true});
        }
    });
})();

(function () {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches || !('IntersectionObserver' in window)) return;
    const sel = '.sec-h, .case, .bento .tile, .gh-stats .tile, .heat-panel, .pgrp, .rev, .shot, .ct, .contact .mega';
    const vh = window.innerHeight;
    const io = new IntersectionObserver(function (entries) {
        entries.forEach(function (e) {
            if (!e.isIntersecting) return;
            e.target.classList.add('in');
            io.unobserve(e.target);
        });
    }, {threshold: 0.12, rootMargin: '0px 0px -6% 0px'});
    document.querySelectorAll(sel).forEach(function (el) {
        const r = el.getBoundingClientRect();
        if (r.top < vh * 0.9) return;
        const sib = el.parentElement ? Array.prototype.indexOf.call(el.parentElement.children, el) : 0;
        el.style.transitionDelay = Math.min(sib, 5) * 70 + 'ms';
        el.classList.add('rv');
        io.observe(el);
    });

    const cio = new IntersectionObserver(function (entries) {
        entries.forEach(function (e) {
            if (!e.isIntersecting) return;
            cio.unobserve(e.target);
            const el = e.target, final = el.textContent.trim();
            const target = parseInt(final.replace(/[^0-9]/g, ''), 10);
            if (!isFinite(target)) return;
            const comma = final.indexOf(',') !== -1, suffix = final.replace(/[0-9,]/g, '');
            const t0 = performance.now(), dur = 1000;

            function fmt(v) {
                return (comma ? v.toLocaleString() : String(v)) + suffix;
            }

            function step(t) {
                const p = Math.min(1, (t - t0) / dur), v = Math.round(target * (1 - Math.pow(1 - p, 3)));
                el.textContent = fmt(v);
                if (p < 1) requestAnimationFrame(step); else el.textContent = final;
            }

            el.textContent = fmt(0);
            requestAnimationFrame(step);
        });
    }, {threshold: 0.6});
    document.querySelectorAll('[data-count]').forEach(function (el) {
        cio.observe(el);
    });
})();
