function initTagInput(containerId, initialTags) {
    const container = document.getElementById(containerId);
    if (!container) {
        return;
    }

    const input = container.querySelector('.tag-text-input');
    const addBtn = container.querySelector('.tag-add-btn');
    const list = container.querySelector('.tag-list');
    const hiddenContainer = container.querySelector('.tag-hidden-inputs');
    let tags = Array.isArray(initialTags) ? [...initialTags] : [];

    function render() {
        list.innerHTML = '';
        hiddenContainer.innerHTML = '';

        tags.forEach((tag, index) => {
            const chip = document.createElement('span');
            chip.className = 'tag-chip';

            const label = document.createElement('span');
            label.textContent = tag;
            chip.appendChild(label);

            const removeBtn = document.createElement('button');
            removeBtn.type = 'button';
            removeBtn.className = 'tag-remove';
            removeBtn.setAttribute('aria-label', 'Remove tag');
            removeBtn.textContent = '×';
            removeBtn.addEventListener('click', () => {
                tags.splice(index, 1);
                render();
            });
            chip.appendChild(removeBtn);

            list.appendChild(chip);

            const hidden = document.createElement('input');
            hidden.type = 'hidden';
            hidden.name = 'tags';
            hidden.value = tag;
            hiddenContainer.appendChild(hidden);
        });
    }

    function addTag() {
        const value = input.value.trim();
        if (!value) {
            return;
        }

        const exists = tags.some((tag) => tag.toLowerCase() === value.toLowerCase());
        if (!exists) {
            tags.push(value);
        }

        input.value = '';
        render();
        input.focus();
    }

    addBtn.addEventListener('click', addTag);
    input.addEventListener('keydown', (event) => {
        if (event.key === 'Enter') {
            event.preventDefault();
            addTag();
        }
    });

    render();
}
