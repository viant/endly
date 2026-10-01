#!/usr/bin/env python3
"""Generate service skills from registered service source and maintained documentation."""
from pathlib import Path
import re
import json
root = Path(__file__).resolve().parents[1]
for source in sorted((root / 'service').rglob('service.go')):
    text = source.read_text()
    match = re.search(r'\bServiceID\s*=\s*"([^"]+)"', text)
    if not match:
        continue
    service = match[1]
    name = 'endly-' + service.replace('/', '-')
    folder = root / 'skills' / name
    folder.mkdir(exist_ok=True)
    action_source = text + '\n' + (source.parent / 'routes.go').read_text() if (source.parent / 'routes.go').exists() else text
    actions = sorted(set(re.findall(r'\bAction:\s*"([^"]+)"', action_source) + re.findall(r's\.route\("([^"]+)"', action_source)))
    description = f'Author and run Endly {service} service actions and diagnose their requests, responses, and workflow integration.'
    body = f'''---
name: {name}
description: {json.dumps(description)}
---

Use this skill for `{service}` actions in Endly YAML workflows.
Inspect the installed contract with `endly -s={service}` and
`endly -s={service} -a=<action>` before adding fields; these commands only print metadata.
Use `action: {service}:<action>` in a pipeline task. Expand state with `$variable`
or `${{variable}}`; use `init` for inputs and `post` to publish results for later tasks.

'''
    if actions:
        body += 'Source-declared actions: ' + ', '.join(f'`{a}`' for a in actions) + '.\n\n'
    readme = source.parent / 'README.md'
    if readme.exists():
        references = folder / 'references'
        references.mkdir(exist_ok=True)
        (references / 'service.md').write_text('# ' + service + ' contract discovery\n\nUse endly_service_info with service and optional action through MCP. Or inspect the live CLI contract with `endly -s=' + service + ' -a=<action>`. Read `' + str(readme.relative_to(root)) + '` in a matching Endly checkout for maintained service documentation.\n\nAvailable source-declared actions: ' + ', '.join(actions) + '.\n')
        body += 'Read [the service reference](references/service.md) for live contract discovery and maintained source locations.\n\n'
    if service == 'dsunit':
        body += 'Function skills: ' + ', '.join('`endly-dsunit-' + x + '`' for x in ('register', 'sequences', 'prepare', 'expect', 'query', 'export', 'mapping', 'hydration', 'batch-validation')) + '. Retrieve only those relevant to the task.\n\n'
    body += f'Source: `{source.relative_to(root)}`. Treat the live contract as authoritative when documentation differs.\n'
    body += 'Run only the actions and environments covered by the user request. A skill documents execution; it does not grant additional authorization. For workflow selection and test design, retrieve the `endly-authoring` and `endly-testing` skills from the same catalog.\n'
    (folder / 'SKILL.md').write_text(body)
