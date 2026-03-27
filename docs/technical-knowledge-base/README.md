# Spiderweb Technical Knowledge Base

A comprehensive, interactive web-based documentation system for the Spiderweb AI Agent System.

## Overview

The Spiderweb Technical Knowledge Base is a modern, responsive documentation website that provides comprehensive technical information about the Spiderweb system architecture, APIs, services, and deployment.

## Features

### 📚 Complete Documentation Coverage
- **System Architecture** - Network topology, security zones, and component relationships
- **API Endpoints** - Complete REST API documentation with examples
- **WebSocket Connections** - Real-time communication protocols and examples
- **Serverless Services** - Trigger.dev and E2B integration details
- **Agent Services** - Agent lifecycle, communication protocols, and management
- **Database Schema** - PostgreSQL and Redis schema definitions
- **Security Infrastructure** - Authentication, authorization, and security measures
- **Monitoring & Metrics** - Prometheus metrics and health checks
- **Integrations** - External service integrations and webhooks
- **Configuration Management** - Environment variables and configuration hierarchy

### 🎨 Modern User Experience
- **Responsive Design** - Works perfectly on desktop, tablet, and mobile devices
- **Dark/Light Theme** - Toggle between themes for comfortable reading
- **Search Functionality** - Real-time search across all documentation content
- **Smooth Navigation** - Smooth scrolling and active navigation highlighting
- **Print Support** - Optimized for printing documentation sections
- **Accessibility** - Full keyboard navigation and screen reader support

### 🔧 Interactive Features
- **Code Copy Buttons** - Copy code examples with a single click
- **Live Search** - Filter content and navigation in real-time
- **Smooth Transitions** - Polished animations and transitions
- **Scroll-to-Top** - Easy navigation with floating action button
- **URL Anchors** - Direct linking to specific sections

## Quick Start

### Viewing the Documentation

1. **Local Viewing**: Simply open `index.html` in any modern web browser
2. **Web Server**: For best results, serve the files through a web server:
   ```bash
   # Using Python 3
   python -m http.server 8000
   
   # Using Node.js
   npx http-server
   
   # Using PHP
   php -S localhost:8000
   ```
3. **Navigate to**: `http://localhost:8000` (or your chosen port)

### Using the Documentation

- **Search**: Use the search box in the sidebar to find specific content
- **Navigation**: Click sidebar links to jump to sections
- **Themes**: Toggle between light and dark themes using the theme button
- **Printing**: Use the print button to print specific sections
- **Code Examples**: Click copy buttons on code blocks to copy to clipboard

## File Structure

```
docs/technical-knowledge-base/
├── index.html          # Main documentation page
├── styles.css          # Complete CSS styling
├── scripts.js          # JavaScript functionality
├── README.md           # This file
└── assets/             # Additional assets (if needed)
```

## Browser Support

The knowledge base supports all modern browsers:
- Chrome 80+
- Firefox 75+
- Safari 13+
- Edge 80+
- Mobile browsers (iOS Safari, Chrome Mobile)

## Technical Details

### Technologies Used
- **HTML5** - Semantic markup and structure
- **CSS3** - Modern styling with CSS Grid and Flexbox
- **JavaScript ES6+** - Interactive functionality and DOM manipulation
- **Prism.js** - Code syntax highlighting (loaded externally)
- **Font Awesome** - Icons (loaded externally)

### Performance Features
- **Lazy Loading** - Optimized loading for better performance
- **Debounced Search** - Efficient search implementation
- **CSS Transitions** - Hardware-accelerated animations
- **Minimal Dependencies** - Only external dependencies are for code highlighting and icons

### Accessibility Features
- **Keyboard Navigation** - Full keyboard support
- **Screen Reader Support** - Proper ARIA labels and semantic HTML
- **High Contrast** - Good color contrast ratios
- **Focus Indicators** - Clear focus states for keyboard users
- **Skip Links** - Skip to main content functionality

## Customization

### Adding New Content
1. Edit `index.html` to add new sections
2. Use the existing section structure as a template
3. Update the sidebar navigation to include new sections
4. Ensure proper heading hierarchy (h2, h3, h4)

### Styling Changes
1. Modify `styles.css` for visual changes
2. Use CSS custom properties (variables) for consistent theming
3. Test changes across different screen sizes
4. Maintain accessibility standards

### JavaScript Enhancements
1. Edit `scripts.js` for new functionality
2. Follow existing code patterns and naming conventions
3. Ensure compatibility with existing features
4. Test thoroughly across browsers

## Port Configuration

The documentation reflects the Spiderweb system's port allocation using the 1337x range:

| Port | Service | Purpose |
|------|---------|---------|
| 13370 | Dashboard Frontend | React development server |
| 13371 | API Gateway | REST API endpoints |
| 13372 | WebSocket Gateway | Real-time communication |
| 13373-13387 | Internal Services | Agent, Provider, Event services |
| 13388 | PostgreSQL | Primary database |
| 13389 | Redis | Message queue and caching |
| 13390+ | File Storage | Static file serving |

## Security Considerations

- **Static Files** - No server-side processing, reducing attack surface
- **CSP Ready** - Can be configured with Content Security Policy
- **No External Dependencies** - Only loads Prism.js and Font Awesome from CDNs
- **HTTPS Compatible** - Works with both HTTP and HTTPS

## Troubleshooting

### Common Issues

**Search not working**: Ensure JavaScript is enabled in your browser
**Styles not loading**: Check that `styles.css` is in the same directory as `index.html`
**Scripts not working**: Verify `scripts.js` is accessible and JavaScript is enabled
**Code highlighting not working**: Check internet connection for external Prism.js loading

### Browser Console
Open browser developer tools to check for any errors or warnings that might indicate issues.

## Contributing

When contributing to this documentation:

1. **Maintain Consistency** - Follow existing formatting and structure patterns
2. **Test Thoroughly** - Test changes across different browsers and devices
3. **Accessibility First** - Ensure all changes maintain accessibility standards
4. **Performance Aware** - Keep the documentation lightweight and fast-loading

## License

This documentation is part of the Spiderweb project. Please refer to the main project license for usage terms.

## Support

For issues related to the technical content, please refer to the main Spiderweb repository. For issues with the documentation website itself, you can:

- Check the browser console for JavaScript errors
- Verify all files are in the correct location
- Test with different browsers to isolate issues
- Ensure your web server is properly configured

## Version Information

- **Last Updated**: March 2026
- **Compatible Browsers**: Modern browsers with ES6+ support
- **File Size**: ~150KB total (HTML + CSS + JS)
- **Dependencies**: Prism.js and Font Awesome (loaded externally)

---

For more information about the Spiderweb system, visit the main project repository or check out the other documentation files in the `docs/` directory.