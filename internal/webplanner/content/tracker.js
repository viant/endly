(function () {
    function handleEvent(e) {
        console.log('Event:', e);
        let holder = e.target.parentElement;
        let i = 0;

        let expectTable = false;
        for (let i = 0; i < 10; i++) {
            if (holder.tagName === 'BODY' || holder.parentElement === null || holder.parentElement.tagName === 'BODY') {
                break;
            }
            if (holder.tagName === 'TD' || holder.tagName === 'TH' || holder.tagName === 'TR') {
                expectTable = true;
            }
            if (holder.tagName === 'TABLE') {
                expectTable = false;
            }
            if (i > 4 && !expectTable) {
                break;
            }
            holder = holder.parentElement;
        }


        console.log('Holder:', holder.outerHTML)
        console.log(holder)


        const eventData = {
            type: e.type,
            targetTag: e.target.tagName,
            timestamp: Date.now(),
            targetHTML: e.target.outerHTML,
            holderHTML: holder.outerHTML,
            key: e.key,
            metaKey: e.metaKey

        };

        const URL = `http://localhost:${port}/event`;
        // Send event data to a server
        fetch(URL, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(eventData),

        })
            .then(response => response.json())
            .then(data => console.log('Event data sent:', data))
            .catch(error => console.error('Error sending event data:', error));
    }

    document.addEventListener('click', handleEvent);
    document.addEventListener('keyup', handleEvent);
    return true;
})();
