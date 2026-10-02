// Проверка типов (vue-tsc) с «базовой линией»: код уже содержит ошибки, поэтому шаг падает
// только если их стало БОЛЬШЕ, чем записано в typecheck-baseline.json (Q-06, ADR-045).
// Стало меньше — поправьте число в baseline: `node scripts/typecheck.mjs --update`.
import { spawnSync } from 'node:child_process';
import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const baselinePath = fileURLToPath(new URL('./typecheck-baseline.json', import.meta.url));
const update = process.argv.includes('--update');

const res = spawnSync('npx', ['vue-tsc', '--noEmit'], { encoding: 'utf8', shell: true });
if (res.error) {
  console.error('Не удалось запустить vue-tsc:', res.error.message);
  process.exit(2);
}
const out = `${res.stdout || ''}${res.stderr || ''}`;
const errors = out.split(/\r?\n/).filter((l) => /error TS\d+/.test(l));
const count = errors.length;

if (count === 0 && res.status !== 0) {
  console.error(out);
  console.error('vue-tsc завершился с ошибкой, но строк «error TS» нет — проверьте вывод выше.');
  process.exit(2);
}

if (update) {
  writeFileSync(baselinePath, `${JSON.stringify({ errors: count }, null, 2)}\n`);
  console.log(`Базовая линия обновлена: ${count}`);
  process.exit(0);
}

const baseline = JSON.parse(readFileSync(baselinePath, 'utf8')).errors;
console.log(`Ошибок типов: ${count} (базовая линия: ${baseline})`);
if (count > baseline) {
  console.error(errors.join('\n'));
  console.error(`\nОшибок стало больше (+${count - baseline}). Исправьте новые.`);
  process.exit(1);
}
if (count < baseline) {
  console.log(`Ошибок меньше, чем в базовой линии: выполните «node scripts/typecheck.mjs --update», чтобы зафиксировать ${count}.`);
}
