"""Verify built metadata and safe source-owned page identity."""

from html.parser import HTMLParser
import json
from pathlib import Path
import tempfile
import unittest

from mkdocs.commands.build import build
from mkdocs.config import load_config
from mkdocs.structure.files import get_files

ROOT = Path(__file__).resolve().parents[1]


class Metadata(HTMLParser):
    def __init__(self, path):
        super().__init__(convert_charrefs=True)
        self.values, self.schemas = {}, []
        self.capture, self.buffer, self.title = None, '', ''
        self.feed(path.read_text(encoding='utf-8'))

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag == 'meta':
            self.values[attrs.get('property') or attrs.get('name')] = attrs.get('content')
        elif tag == 'link' and attrs.get('rel') == 'canonical':
            self.values['canonical'] = attrs['href']
        elif tag == 'title' or (tag == 'script' and attrs.get('type') == 'application/ld+json'):
            self.capture, self.buffer = tag, ''

    def handle_data(self, data):
        if self.capture:
            self.buffer += data

    def handle_endtag(self, tag):
        if tag == self.capture:
            if tag == 'title':
                self.title = self.buffer
            else:
                self.schemas.append(json.loads(self.buffer))
            self.capture = None


class DocumentationSchemaTests(unittest.TestCase):
    def check_identity(self, path):
        page = Metadata(path)
        schemas = [item for item in page.schemas if item.get('@type') in ('TechArticle', 'WebPage')]
        self.assertEqual(len(schemas), 1, str(path))
        schema = schemas[0]
        self.assertEqual(schema['name'], page.title)
        self.assertEqual(schema['name'], page.values['og:title'])
        self.assertEqual(schema['description'], page.values['description'])
        self.assertEqual(schema['description'], page.values['og:description'])
        self.assertEqual(schema['url'], page.values['canonical'])
        self.assertEqual(schema['@id'], page.values['og:url'])
        self.assertFalse(schema['url'].endswith('/'))
        return schema

    def test_published_documentation(self):
        config = load_config(config_file=str(ROOT / 'mkdocs.yml'))
        checked = 0
        for source in get_files(config).documentation_pages():
            path = Path(config.site_dir) / source.dest_uri
            with self.subTest(source=source.src_uri):
                page = Metadata(path)
                if any(item.get('@type') in ('TechArticle', 'WebPage') for item in page.schemas):
                    self.check_identity(path)
                    checked += 1
        self.assertGreater(checked, 1)

    def test_authored_punctuation_and_encoded_canonical(self):
        with tempfile.TemporaryDirectory(dir=ROOT) as workspace:
            root = Path(workspace)
            docs = root / 'docs'
            docs.mkdir()
            source = 'Café & space.md'
            (docs / source).write_text(
                '---\ntitle: \'C# & "types"\'\n'
                'description: \'Use "quotes" & </script> safely.\'\n---\n'
                '# Visible heading\n\nOriginal prose.\n', encoding='utf-8')
            config = load_config(
                config_file=str(ROOT / 'mkdocs.yml'), docs_dir=str(docs),
                site_dir=str(root / 'site'), site_url='https://example.test/docs%20root/',
                nav=[{'Quoted "section"': [source]}])
            build(config)
            output = root / 'site' / source[:-3] / 'index.html'
            schema = self.check_identity(output)
            self.assertEqual(schema['url'], 'https://example.test/docs%20root/Caf%C3%A9%20%26%20space')
            self.assertEqual(schema['description'], 'Use "quotes" & </script> safely.')
            self.assertIn('Original prose.', output.read_text(encoding='utf-8'))


if __name__ == '__main__':
    unittest.main()
