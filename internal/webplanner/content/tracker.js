(function () {
    if (window.__endlyLiveRecorder) {
        return true;
    }

    const endpoint = "http://${host}:${port}/event?token=${token}";
    const pendingInputs = new Map();
    const lastValues = new WeakMap();
    let sequence = 0;

    function holderFor(target) {
        let holder = target && target.parentElement;
        let tableContext = false;
        for (let depth = 0; holder && depth < 10; depth++) {
            if (holder.tagName === "TD" || holder.tagName === "TH" || holder.tagName === "TR") {
                tableContext = true;
            }
            if (holder.tagName === "TABLE") {
                tableContext = false;
            }
            if (holder.tagName === "BODY" || !holder.parentElement || holder.parentElement.tagName === "BODY") {
                break;
            }
            if (depth > 4 && !tableContext) {
                break;
            }
            holder = holder.parentElement;
        }
        return holder;
    }

    function boundedHTML(element) {
        if (!element || !element.outerHTML) {
            return "";
        }
        const clone = element.cloneNode(true);
        const sensitive = [];
        if (clone.matches && clone.matches('input[type="password"]')) {
            sensitive.push(clone);
        }
        if (clone.querySelectorAll) {
            clone.querySelectorAll('input[type="password"]').forEach(function (input) { sensitive.push(input); });
        }
        sensitive.forEach(function (input) {
            input.removeAttribute("value");
            input.setAttribute("data-endly-redacted", "true");
        });
        return clone.outerHTML.slice(0, 100000);
    }

    function valueFor(target) {
        if (!target) {
            return {value: "", redacted: false};
        }
        const inputType = String(target.type || "").toLowerCase();
        if (inputType === "password") {
            return {value: "", redacted: true};
        }
        if (inputType === "checkbox" || inputType === "radio") {
            return {value: String(Boolean(target.checked)), redacted: false};
        }
        return {value: String(target.value || ""), redacted: false};
    }

    function send(type, target, event) {
        const holder = holderFor(target);
        const captured = valueFor(target);
        if (target && (type === "input" || type === "change")) {
            const signature = `${captured.redacted}:${captured.value}:${Boolean(target.checked)}`;
            if (lastValues.get(target) === signature) {
                return;
            }
            lastValues.set(target, signature);
        }
        const payload = {
            id: `${Date.now()}-${++sequence}-${Math.random().toString(36).slice(2)}`,
            type: type,
            targetTag: target ? target.tagName : "",
            timestamp: Date.now(),
            targetHTML: boundedHTML(target),
            holderHTML: boundedHTML(holder),
            key: event && event.key ? event.key : "",
            metaKey: Boolean(event && event.metaKey),
            value: captured.value,
            valueRedacted: captured.redacted,
            inputType: target ? String(target.type || "") : "",
            checked: Boolean(target && target.checked),
            url: window.location.href,
            title: document.title
        };
        try {
            const encoded = btoa(unescape(encodeURIComponent(JSON.stringify(payload))));
            console.debug("__ENDLY_EVENT_B64__" + encoded);
        } catch (_) {}
        fetch(endpoint, {
            method: "POST",
            headers: {"Content-Type": "application/json"},
            body: JSON.stringify(payload),
            keepalive: true
        }).catch(function () {});
    }

    function onInput(event) {
        const target = event.target;
        const previous = pendingInputs.get(target);
        if (previous) {
            clearTimeout(previous);
        }
        pendingInputs.set(target, setTimeout(function () {
            pendingInputs.delete(target);
            send("input", target, event);
        }, 300));
    }

    function onChange(event) {
        const previous = pendingInputs.get(event.target);
        if (previous) {
            clearTimeout(previous);
            pendingInputs.delete(event.target);
        }
        send("change", event.target, event);
    }

    function onKeyDown(event) {
        if (["Enter", "Escape"].includes(event.key)) {
            send("press", event.target, event);
        }
    }

    function onNavigation() {
        send("navigation", null, null);
    }

    document.addEventListener("click", function (event) {
        if (event.target && event.target.matches && event.target.matches("input, textarea, select, option, label")) {
            return;
        }
        send("click", event.target, event);
    }, true);
    document.addEventListener("input", onInput, true);
    document.addEventListener("change", onChange, true);
    document.addEventListener("keydown", onKeyDown, true);
    window.addEventListener("popstate", onNavigation);
    window.addEventListener("hashchange", onNavigation);

    window.__endlyLiveRecorder = true;
    send("navigation", null, null);
    return true;
})();
