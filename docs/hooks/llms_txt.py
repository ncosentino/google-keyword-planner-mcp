"""MkDocs build hook: auto-generate llms.txt at build time.

Walks the site nav, collects each page's title and description (from front
matter or the global site_description fallback), then writes site/llms.txt
following the llmstxt.org convention. The file stays in sync automatically
as pages are added, removed, or re-described.
"""

import os
from pathlib import Path


_PREAMBLE = """\
# Google Keyword Planner MCP

> Zero-dependency MCP server that exposes Google Ads Keyword Planner to AI assistants.

The Google Keyword Planner MCP server provides native binaries for Go and C#.
It exposes three MCP tools -- generate_keyword_ideas, get_historical_metrics, get_keyword_forecast --
directly to AI assistants like Claude, GitHub Copilot, and Cursor.
Author: Nick Cosentino (https://www.devleader.ca).

"""

_PAGE_DESCRIPTIONS: dict[str, str] = {}


def on_page_context(context, page, config, nav) -> None:
    """Collect each page's description as pages are processed."""
    desc = ''
    if page.meta and page.meta.get('description'):
        desc = page.meta['description']
    else:
        desc = config.get('site_description', '')
    _PAGE_DESCRIPTIONS[page.url or ''] = (page.title or '', desc)


def on_post_build(config) -> None:
    """Write site/llms.txt after the full build completes."""
    site_url = config.get('site_url', '').rstrip('/')
    nav = config.get('nav', [])

    lines: list[str] = [_PREAMBLE]

    def _section_header(title: str) -> str:
        return f'\n## {title}\n\n'

    def _page_line(url: str) -> str:
        url_clean = url.strip('/')
        if url_clean:
            full_url = f'{site_url}/{url_clean}/'
        else:
            full_url = f'{site_url}/'
        title, desc = _PAGE_DESCRIPTIONS.get(url, (url, ''))
        if desc:
            return f'- [{title}]({full_url}) -- {desc}\n'
        return f'- [{title}]({full_url})\n'

    def _walk_nav(nav_items, indent: int = 0) -> None:
        for item in nav_items:
            if isinstance(item, dict):
                for section_title, children in item.items():
                    if isinstance(children, list):
                        lines.append(_section_header(section_title))
                        _walk_nav(children, indent + 1)
                    elif isinstance(children, str):
                        page_url = _url_from_path(children)
                        if page_url is not None:
                            lines.append(_page_line(page_url))
            elif isinstance(item, str):
                page_url = _url_from_path(item)
                if page_url is not None:
                    lines.append(_page_line(page_url))

    def _url_from_path(md_path: str) -> str:
        """Convert a docs-relative .md path to its URL segment."""
        if md_path.lower().endswith('readme.md'):
            return None
        url = md_path.replace('.md', '/')
        if url == 'index/':
            url = ''
        url = url.replace('index/', '')
        return url

    _walk_nav(nav)

    output_path = Path(config['site_dir']) / 'llms.txt'
    output_path.write_text(''.join(lines), encoding='utf-8')
