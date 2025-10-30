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
