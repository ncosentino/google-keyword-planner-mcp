"""MkDocs build hooks for structured data injection."""

import json
import re
from copy import copy
from urllib.parse import quote, unquote, urlsplit, urlunsplit

from markupsafe import escape


def on_page_markdown(markdown, page, config, files):
    if page.canonical_url:
        url = urlsplit(config.site_url)
        path = url.path if page.is_homepage else url.path.rstrip('/') + '/' + page.url
        path = '/'.join(quote(unquote(part), safe='') for part in path.split('/'))
        page.canonical_url = urlunsplit(url._replace(path=path.rstrip('/')))
    return markdown


def on_page_context(context, page, config, nav):
    title = page.meta.get('title') or page.title
    title = config.site_name if page.is_homepage else f'{title} - {config.site_name}'
    context['documentation_title'] = title
    description = page.meta.get('description')
    context['documentation_schema'] = ''
    if description:
        if not isinstance(description, str):
            raise ValueError(f'{page.file.src_uri}: description must be a string.')
        schema = {
            '@context': 'https://schema.org',
            '@type': 'WebPage',
            '@id': page.canonical_url,
            'url': page.canonical_url,
            'name': title,
            'description': description,
        }
        payload = json.dumps(schema, ensure_ascii=False).replace('<', '\\u003c')
        context['documentation_schema'] = (
            '<script type="application/ld+json">' + payload + '</script>'
        )
        rendered_page = copy(context['page'])
        rendered_page.meta = {**rendered_page.meta, 'description': escape(description)}
        context['page'] = rendered_page
    return context


def _extract_h2_sections(html: str) -> list[tuple[str, str]]:
    """Return a list of (heading_text, first_paragraph_text) for each H2 in html."""
    sections = []
    pattern = re.compile(
        r'<h2[^>]*>(.*?)</h2>(.*?)(?=<h2|$)',
        re.DOTALL | re.IGNORECASE,
    )
    for match in pattern.finditer(html):
        raw_heading = match.group(1)
        heading = re.sub(r'<[^>]+>', '', raw_heading)
        heading = re.sub(r'&[a-z#0-9]+;', '', heading).strip()
        body_html = match.group(2)
        first_para = re.search(r'<p>(.*?)</p>', body_html, re.DOTALL)
        para_text = re.sub(r'<[^>]+>', '', first_para.group(1)).strip() if first_para else ''
        para_text = re.sub(r'\s+', ' ', para_text)
        if heading:
            sections.append((heading, para_text))
    return sections


def on_page_content(html: str, page, config, files) -> str:
    """Inject structured data JSON-LD blocks based on page URL."""
    url = page.url or ''

    if url == 'getting-started/':
        html = _inject_howto_schema(html, page)

    if url.startswith('tools/') and url != 'tools/':
        html = _inject_faq_schema(html, page)

    return html


def _inject_howto_schema(html: str, page) -> str:
    """Inject HowTo JSON-LD for the Getting Started page."""
    sections = _extract_h2_sections(html)
    if not sections:
        return html

    steps = [
        {
            '@type': 'HowToStep',
            'name': heading,
            'text': para_text,
        }
        for heading, para_text in sections
        if heading.lower() not in ('next steps', 'see also')
    ]

    schema = {
        '@context': 'https://schema.org',
        '@type': 'HowTo',
        'name': page.title,
        'description': (
            page.meta.get('description', '')
            if page.meta
            else ''
        ),
        'step': steps,
    }

    script_tag = (
        '<script type="application/ld+json">\n'
        + json.dumps(schema, indent=2, ensure_ascii=False)
        + '\n</script>\n'
    )
    return html + script_tag


def _inject_faq_schema(html: str, page) -> str:
    """Inject FAQPage JSON-LD for MCP tool reference pages."""
    tool_name = (page.title or '').strip()
    sections = _extract_h2_sections(html)

    questions = []
    for heading, para_text in sections:
        h = heading.lower()
        if h in ('overview', 'see also', 'notes'):
            continue
        if not para_text:
            continue
        questions.append({
            '@type': 'Question',
            'name': f'{heading} for {tool_name}?',
            'acceptedAnswer': {'@type': 'Answer', 'text': para_text},
        })

    if not questions:
        return html

    schema = {
        '@context': 'https://schema.org',
        '@type': 'FAQPage',
        'mainEntity': questions,
    }

    script_tag = (
        '<script type="application/ld+json">\n'
        + json.dumps(schema, indent=2, ensure_ascii=False)
        + '\n</script>\n'
    )
    return html + script_tag
