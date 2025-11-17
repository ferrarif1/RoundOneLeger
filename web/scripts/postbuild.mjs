import { cp, mkdir, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';

const distDir = join(process.cwd(), 'dist');
await mkdir(distDir, { recursive: true });
const placeholder = `This placeholder keeps the dist/ directory under version control so the Go
build always has something to embed. Run \`npm run build\` inside \`web/\` to
replace the contents with the real production bundle.`;
await writeFile(join(distDir, 'README.md'), `${placeholder}\n`, 'utf8');

const embedDir = join(process.cwd(), '..', 'webembed', 'dist');
await rm(embedDir, { recursive: true, force: true });
await mkdir(embedDir, { recursive: true });
await cp(distDir, embedDir, { recursive: true });
