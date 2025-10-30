// Dark mode toggle
function toggleDarkMode() {
    document.body.classList.toggle('dark-mode');
    const isDark = document.body.classList.contains('dark-mode');
    localStorage.setItem('darkMode', isDark);
    updateDarkModeButton(isDark);
}

function updateDarkModeButton(isDark) {
    const btn = document.getElementById('darkModeToggle');
    btn.textContent = isDark ? '☀️ Light Mode' : '🌙 Dark Mode';
}

// Carregar preferência salva
window.addEventListener('DOMContentLoaded', () => {
    // Se não houver preferência salva, usa dark mode como padrão
    const savedMode = localStorage.getItem('darkMode');
    const isDark = savedMode === null ? true : savedMode === 'true';
    
    if (isDark) {
        document.body.classList.add('dark-mode');
    }
    updateDarkModeButton(isDark);
});

// Function to render title with version (called from Go)
window.renderTitleVersion = function(version) {
    const titleElement = document.getElementById('title');
    if (titleElement && version) {
        titleElement.setAttribute('data-version', version);
        titleElement.textContent = `🚀 HPA Demo v${version}`;
    }
};

// Function to update legend values (called from Go)
window.updateLegend = function(legendPrefix, min, max, current) {
    const minEl = document.getElementById(`${legendPrefix}_min`);
    const maxEl = document.getElementById(`${legendPrefix}_max`);
    const currentEl = document.getElementById(`${legendPrefix}_current`);
    
    if (minEl) minEl.textContent = min;
    if (maxEl) maxEl.textContent = max;
    if (currentEl) currentEl.textContent = current;
};
