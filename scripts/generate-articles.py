#!/usr/bin/env python3
"""
Generate articles.md from the Dev Leader Google Keyword Planner MCP tag page.
Usage: python scripts/generate-articles.py docs/articles.md
"""

import sys
import urllib.request
from html.parser import HTMLParser
from datetime import datetime

_VOID_ELEMENTS = frozenset({
    'area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input',
    'link', 'meta', 'param', 'source', 'track', 'wbr',
})

_TAG_URL = 'https://www.devleader.ca/tags/GoogleKeywordPlannerMcp'


class _ArticleParser(HTMLParser):
    """Parse tagged articles from the devleader.ca tags page."""

    def __init__(self):
        super().__init__()
        self.articles = []
        self._cur = None
        self._depth = 0
        self._section = None
        self._section_depth = 0
        self._state = None

    def _attrs_dict(self, attrs):
        return {k: (v or '') for k, v in attrs}

    def handle_starttag(self, tag, attrs):
        if tag not in _VOID_ELEMENTS:
            self._depth += 1
        a = self._attrs_dict(attrs)
        classes = a.get('class', '')

        if tag == 'article':
            self._cur = {'title': '', 'link': '', 'description': '', 'date': '', 'image': None, 'categories': []}
            self._section = None
            self._state = None
            return

        if self._cur is None:
            return

        if self._section is None:
            if tag == 'div' and 'photo' in classes:
                self._section = 'photo'
                self._section_depth = self._depth
            elif tag == 'li' and 'date' in classes:
                self._section = 'date'
                self._section_depth = self._depth
            elif tag == 'li' and 'tags' in classes:
                self._section = 'tags'
                self._section_depth = self._depth
            elif tag == 'div' and 'description' in classes:
                self._section = 'description'
                self._section_depth = self._depth
            return

        if self._section == 'photo' and tag == 'img':
            self._cur['image'] = a.get('src')
        elif self._section == 'date' and tag == 'span' and self._state is None:
            self._state = 'date_span'
        elif self._section == 'tags' and tag == 'a' and 'goto-tag' in classes:
            self._state = 'tag_a'
        elif self._section == 'description':
            if tag == 'h2' and self._state is None:
                self._state = 'h2'
            elif tag == 'p' and 'read-more' in classes:
                self._state = 'read_more'
            elif tag == 'a' and self._state == 'read_more':
                href = a.get('href', '')
                if href.startswith('/'):
                    self._cur['link'] = 'https://www.devleader.ca' + href
            elif tag == 'p' and self._state is None:
                self._state = 'desc_p'

    def handle_endtag(self, tag):
        self._depth -= 1

        if tag == 'article':
            if self._cur and self._cur['title'] and self._cur['link']:
                self.articles.append(self._cur)
            self._cur = None
            self._section = None
            self._state = None
            return

        if self._cur is None:
            return

        if self._section is not None and self._depth < self._section_depth:
            self._section = None
            self._state = None
            return

        if self._state == 'date_span' and tag == 'span':
            self._state = None
        elif self._state == 'tag_a' and tag == 'a':
            self._state = None
        elif self._state == 'h2' and tag == 'h2':
            self._state = None
        elif self._state == 'read_more' and tag == 'p':
            self._state = None
        elif self._state == 'desc_p' and tag == 'p':
            self._state = None

    def handle_data(self, data):
        data = data.strip()
        if not data or self._cur is None:
            return

        if self._state == 'h2':
            self._cur['title'] += data
        elif self._state == 'desc_p':
            self._cur['description'] += data
        elif self._state == 'date_span':
            try:
                dt = datetime.strptime(data, '%m/%d/%Y')
                self._cur['date'] = dt.strftime('%B %d, %Y')
            except ValueError:
                self._cur['date'] = data
        elif self._state == 'tag_a':
            self._cur['categories'].append(data)


def fetch_articles(url: str) -> list[dict]:
    """Fetch and parse articles from the Dev Leader tags page."""
    req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
    with urllib.request.urlopen(req, timeout=30) as response:
        html = response.read().decode('utf-8')
    parser = _ArticleParser()
    parser.feed(html)
    return parser.articles


def generate_markdown(articles: list[dict]) -> str:
    """Generate markdown content from articles."""
    lines = [
        "# Articles & Blog Posts",
        "",
        "Blog posts and articles about this MCP server from [Dev Leader](https://www.devleader.ca).",
        "",
        "!!! tip \"Stay Updated\"",
        "    Follow [Dev Leader](https://www.devleader.ca) for articles about MCP servers, Google Ads Keyword Planner, and C#/.NET development.",
        "",
    ]

    if not articles:
        lines.extend([
            "*No articles published yet. Check back soon!*",
            "",
        ])
    else:
        for article in articles:
            if article.get('image'):
                lines.append(f"[![{article['title']}]({article['image']})]({article['link']})")
                lines.append("")

            lines.append(f"## [{article['title']}]({article['link']})")
            lines.append("")
            if article['date']:
                lines.append(f"*Published: {article['date']}*")
                lines.append("")
            if article['description']:
                lines.append(f"> {article['description']}")
                lines.append("")
            if article['categories']:
                tags = ", ".join(f"`{c}`" for c in article['categories'][:5])
                lines.append(f"**Tags:** {tags}")
                lines.append("")
            lines.append("---")
            lines.append("")

    lines.extend([
        "",
        "*This page is automatically generated from the Dev Leader blog.*",
    ])

    return "\n".join(lines)


def main():
    if len(sys.argv) < 2:
        print('Usage: python generate-articles.py <output-file>')
        sys.exit(1)

    output_file = sys.argv[1]

    print(f'Fetching articles from {_TAG_URL}...')

    try:
        articles = fetch_articles(_TAG_URL)
        print(f'Found {len(articles)} articles')
        markdown = generate_markdown(articles)

        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(markdown)

        print(f'Generated {output_file}')

    except Exception as e:
        print(f'Error fetching articles: {e}')
        fallback = generate_markdown([])

        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(fallback)
        print(f'Created fallback {output_file}')


if __name__ == "__main__":
    main()
