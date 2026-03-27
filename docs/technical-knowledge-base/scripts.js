// Knowledge Base JavaScript functionality

document.addEventListener('DOMContentLoaded', function() {
    // Initialize all functionality
    initSidebar();
    initSearch();
    initThemeToggle();
    initScrollTop();
    initNavigation();
    initCodeHighlighting();
    initPrintStyles();
});

// Sidebar functionality
function initSidebar() {
    const sidebar = document.getElementById('sidebar');
    const sidebarToggle = document.getElementById('sidebarToggle');
    const mainContent = document.getElementById('mainContent');

    if (sidebarToggle) {
        sidebarToggle.addEventListener('click', () => {
            sidebar.classList.toggle('active');
            if (sidebar.classList.contains('active')) {
                mainContent.style.marginLeft = '0';
            } else {
                mainContent.style.marginLeft = getComputedStyle(document.documentElement).getPropertyValue('--sidebar-width');
            }
        });
    }

    // Close sidebar on mobile when clicking outside
    document.addEventListener('click', (e) => {
        const isMobile = window.innerWidth <= 900;
        if (isMobile && sidebar.classList.contains('active')) {
            if (!sidebar.contains(e.target) && !sidebarToggle.contains(e.target)) {
                sidebar.classList.remove('active');
                mainContent.style.marginLeft = '0';
            }
        }
    });

    // Handle window resize
    window.addEventListener('resize', () => {
        const isMobile = window.innerWidth <= 900;
        if (!isMobile) {
            sidebar.classList.remove('active');
            mainContent.style.marginLeft = getComputedStyle(document.documentElement).getPropertyValue('--sidebar-width');
        }
    });
}

// Search functionality
function initSearch() {
    const searchInput = document.getElementById('searchInput');
    const navLinks = document.querySelectorAll('.nav-section a');
    const contentSections = document.querySelectorAll('.content-section');

    if (!searchInput) return;

    searchInput.addEventListener('input', (e) => {
        const query = e.target.value.toLowerCase().trim();
        
        if (!query) {
            // Reset all elements when search is empty
            navLinks.forEach(link => {
                link.style.display = 'block';
                link.style.opacity = '1';
            });
            contentSections.forEach(section => {
                section.style.display = 'block';
                section.style.opacity = '1';
            });
            return;
        }

        // Filter navigation links
        let foundInNav = false;
        navLinks.forEach(link => {
            const text = link.textContent.toLowerCase();
            const href = link.getAttribute('href');
            const sectionId = href ? href.replace('#', '') : '';
            
            if (text.includes(query) || sectionId.includes(query)) {
                link.style.display = 'block';
                link.style.opacity = '1';
                foundInNav = true;
            } else {
                link.style.display = 'none';
                link.style.opacity = '0.5';
            }
        });

        // Filter content sections
        contentSections.forEach(section => {
            const sectionId = section.id;
            const sectionText = section.textContent.toLowerCase();
            
            if (sectionText.includes(query) || sectionId.includes(query)) {
                section.style.display = 'block';
                section.style.opacity = '1';
            } else {
                section.style.display = 'none';
                section.style.opacity = '0.5';
            }
        });
    });
}

// Theme toggle functionality
function initThemeToggle() {
    const themeToggle = document.getElementById('themeToggle');
    
    if (!themeToggle) return;

    // Check for saved theme preference
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme) {
        document.documentElement.setAttribute('data-theme', savedTheme);
        updateThemeIcon(themeToggle, savedTheme);
    }

    themeToggle.addEventListener('click', () => {
        const currentTheme = document.documentElement.getAttribute('data-theme');
        const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
        
        document.documentElement.setAttribute('data-theme', newTheme);
        localStorage.setItem('theme', newTheme);
        updateThemeIcon(themeToggle, newTheme);
    });
}

function updateThemeIcon(button, theme) {
    const icon = button.querySelector('i');
    if (icon) {
        if (theme === 'dark') {
            icon.className = 'fas fa-sun';
            icon.style.color = '#f59e0b';
        } else {
            icon.className = 'fas fa-moon';
            icon.style.color = '#64748b';
        }
    }
}

// Scroll to top functionality
function initScrollTop() {
    const scrollTopBtn = document.getElementById('scrollTopBtn');
    
    if (!scrollTopBtn) return;

    // Show/hide button based on scroll position
    window.addEventListener('scroll', () => {
        if (window.pageYOffset > 300) {
            scrollTopBtn.style.opacity = '1';
            scrollTopBtn.style.transform = 'translateY(0)';
        } else {
            scrollTopBtn.style.opacity = '0';
            scrollTopBtn.style.transform = 'translateY(20px)';
        }
    });

    // Smooth scroll to top
    scrollTopBtn.addEventListener('click', () => {
        window.scrollTo({
            top: 0,
            behavior: 'smooth'
        });
    });
}

// Navigation functionality
function initNavigation() {
    const navLinks = document.querySelectorAll('.nav-section a');
    const contentSections = document.querySelectorAll('.content-section');

    // Smooth scrolling for navigation links
    navLinks.forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const targetId = link.getAttribute('href').replace('#', '');
            const targetSection = document.getElementById(targetId);
            
            if (targetSection) {
                // Update active navigation
                navLinks.forEach(l => l.classList.remove('active'));
                link.classList.add('active');

                // Scroll to section
                targetSection.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });

                // Update URL hash
                history.pushState(null, null, `#${targetId}`);
            }
        });
    });

    // Update active navigation on scroll
    window.addEventListener('scroll', () => {
        let current = '';
        const scrollPosition = window.scrollY + 100; // Offset for header

        contentSections.forEach(section => {
            const sectionTop = section.offsetTop;
            const sectionHeight = section.clientHeight;
            
            if (scrollPosition >= sectionTop && scrollPosition < sectionTop + sectionHeight) {
                current = section.getAttribute('id');
            }
        });

        // Update active navigation
        navLinks.forEach(link => {
            link.classList.remove('active');
            if (link.getAttribute('href') === `#${current}`) {
                link.classList.add('active');
            }
        });
    });
}

// Code highlighting
function initCodeHighlighting() {
    // Prism.js is loaded externally, but we can add some custom functionality
    const codeBlocks = document.querySelectorAll('pre code');
    
    codeBlocks.forEach(block => {
        // Add copy button to code blocks
        const copyBtn = document.createElement('button');
        copyBtn.className = 'copy-btn';
        copyBtn.innerHTML = '<i class="fas fa-copy"></i>';
        copyBtn.title = 'Copy to clipboard';
        
        const pre = block.parentElement;
        pre.style.position = 'relative';
        pre.appendChild(copyBtn);

        copyBtn.addEventListener('click', async () => {
            try {
                await navigator.clipboard.writeText(block.textContent);
                copyBtn.innerHTML = '<i class="fas fa-check"></i>';
                copyBtn.style.backgroundColor = '#10b981';
                
                setTimeout(() => {
                    copyBtn.innerHTML = '<i class="fas fa-copy"></i>';
                    copyBtn.style.backgroundColor = '';
                }, 2000);
            } catch (err) {
                console.error('Failed to copy text: ', err);
            }
        });
    });
}

// Print styles enhancement
function initPrintStyles() {
    const printBtn = document.querySelector('.btn-primary');
    
    if (printBtn) {
        printBtn.addEventListener('click', () => {
            window.print();
        });
    }

    // Add print date and URL
    window.addEventListener('beforeprint', () => {
        const footer = document.createElement('div');
        footer.className = 'print-footer';
        footer.innerHTML = `
            <hr style="border-color: #ccc; margin: 20px 0;">
            <div style="font-size: 0.8rem; color: #666; text-align: center;">
                Printed on ${new Date().toLocaleDateString()} from ${window.location.href}
            </div>
        `;
        document.body.appendChild(footer);
    });

    window.addEventListener('afterprint', () => {
        const footer = document.querySelector('.print-footer');
        if (footer) {
            footer.remove();
        }
    });
}

// Accessibility enhancements
function initAccessibility() {
    // Add skip to content link
    const skipLink = document.createElement('a');
    skipLink.href = '#mainContent';
    skipLink.className = 'skip-link';
    skipLink.textContent = 'Skip to main content';
    document.body.insertBefore(skipLink, document.body.firstChild);

    // Keyboard navigation for sidebar
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            const sidebar = document.getElementById('sidebar');
            if (sidebar && sidebar.classList.contains('active')) {
                sidebar.classList.remove('active');
                document.getElementById('mainContent').style.marginLeft = '0';
            }
        }
    });
}

// Performance optimizations
function initPerformance() {
    // Lazy loading for images (if any are added later)
    const lazyImages = document.querySelectorAll('img[loading="lazy"]');
    if ('IntersectionObserver' in window) {
        const imageObserver = new IntersectionObserver((entries, observer) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const img = entry.target;
                    img.src = img.dataset.src;
                    img.removeAttribute('data-src');
                    imageObserver.unobserve(img);
                }
            });
        });

        lazyImages.forEach(img => {
            imageObserver.observe(img);
        });
    }

    // Debounce search input
    function debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }

    const searchInput = document.getElementById('searchInput');
    if (searchInput) {
        searchInput.addEventListener('input', debounce((e) => {
            // Search logic handled in initSearch
        }, 300));
    }
}

// Initialize accessibility and performance
initAccessibility();
initPerformance();

// Export functions for potential external use
window.SpiderwebKB = {
    initSidebar,
    initSearch,
    initThemeToggle,
    initScrollTop,
    initNavigation,
    initCodeHighlighting,
    initPrintStyles
};