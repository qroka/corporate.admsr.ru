import { jsPDF } from 'jspdf';

/** Печать и скачивание PDF-сертификата о прохождении обучения. */

export type CourseCertificatePayload = {
  courseTitle: string;
  userFio: string;
  completedAt?: string | null;
  ofoName?: string | null;
  finalScore?: number | null;
  passed?: boolean;
};

function escapeHtml(s: string): string {
  return String(s || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function formatDate(iso?: string | null): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  const date = d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' });
  const time = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  return `${date}, ${time}`;
}

function safeFileName(title: string): string {
  const base = String(title || 'обучение')
    .replace(/[<>:"/\\|?*\u0000-\u001f]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 80);
  return `Сертификат — ${base || 'обучение'}.pdf`;
}

function wrapText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number): string[] {
  const words = String(text || '').split(/\s+/).filter(Boolean);
  if (!words.length) return [''];
  const lines: string[] = [];
  let line = words[0];
  for (let i = 1; i < words.length; i++) {
    const test = `${line} ${words[i]}`;
    if (ctx.measureText(test).width <= maxWidth) {
      line = test;
    } else {
      lines.push(line);
      line = words[i];
    }
  }
  lines.push(line);
  return lines;
}

/** Рендер без CSS/html2canvas — только canvas (кириллица через системный шрифт). */
function drawCertificateCanvas(payload: CourseCertificatePayload): HTMLCanvasElement {
  const w = 2246;
  const h = 1588;
  const canvas = document.createElement('canvas');
  canvas.width = w;
  canvas.height = h;
  const ctx = canvas.getContext('2d');
  if (!ctx) throw new Error('Canvas недоступен');

  // фон
  ctx.fillStyle = '#ffffff';
  ctx.fillRect(0, 0, w, h);

  // мягкие акценты
  const g1 = ctx.createLinearGradient(0, 0, w * 0.6, h * 0.5);
  g1.addColorStop(0, 'rgba(16, 185, 129, 0.10)');
  g1.addColorStop(1, 'rgba(16, 185, 129, 0)');
  ctx.fillStyle = g1;
  ctx.fillRect(0, 0, w, h);

  const g2 = ctx.createLinearGradient(w, 0, w * 0.4, h * 0.5);
  g2.addColorStop(0, 'rgba(30, 64, 120, 0.08)');
  g2.addColorStop(1, 'rgba(30, 64, 120, 0)');
  ctx.fillStyle = g2;
  ctx.fillRect(0, 0, w, h);

  // рамки
  ctx.strokeStyle = '#1e3a5f';
  ctx.lineWidth = 6;
  ctx.strokeRect(36, 36, w - 72, h - 72);
  ctx.strokeStyle = '#10b981';
  ctx.lineWidth = 2;
  ctx.strokeRect(64, 64, w - 128, h - 128);

  const padX = 140;
  const maxText = w - padX * 2;
  let y = 180;

  ctx.fillStyle = '#5b6b7c';
  ctx.font = '500 28px "Segoe UI", "PT Sans", Arial, sans-serif';
  ctx.fillText('КОРПОРАТИВНЫЙ ПОРТАЛ ADMSR', padX, y);
  y += 100;

  ctx.fillStyle = '#0f2744';
  ctx.font = '600 96px "Segoe UI", "PT Sans", Arial, sans-serif';
  ctx.fillText('Сертификат', padX, y);
  y += 90;

  ctx.fillStyle = '#4b5c6b';
  ctx.font = '400 40px "Segoe UI", "PT Sans", Arial, sans-serif';
  ctx.fillText('Настоящим подтверждается, что', padX, y);
  y += 80;

  ctx.fillStyle = '#0f2744';
  ctx.font = '600 72px "Segoe UI", "PT Sans", Arial, sans-serif';
  for (const line of wrapText(ctx, payload.userFio || 'Сотрудник', maxText)) {
    ctx.fillText(line, padX, y);
    y += 84;
  }
  y += 24;

  ctx.fillStyle = '#5b6b7c';
  ctx.font = '400 34px "Segoe UI", "PT Sans", Arial, sans-serif';
  ctx.fillText('успешно прошёл(а) обучение', padX, y);
  y += 70;

  ctx.fillStyle = '#0b7a55';
  ctx.font = '600 56px "Segoe UI", "PT Sans", Arial, sans-serif';
  const courseLines = wrapText(ctx, `«${payload.courseTitle || 'Обучение'}»`, maxText);
  for (const line of courseLines) {
    ctx.fillText(line, padX, y);
    y += 68;
  }
  y += 50;

  const meta: { label: string; value: string }[] = [
    { label: 'ДАТА ЗАВЕРШЕНИЯ', value: formatDate(payload.completedAt) },
  ];
  if (payload.ofoName) {
    meta.push({ label: 'ПОДРАЗДЕЛЕНИЕ', value: String(payload.ofoName) });
  }
  if (payload.finalScore != null && Number.isFinite(Number(payload.finalScore))) {
    meta.push({ label: 'ИТОГОВЫЙ РЕЗУЛЬТАТ', value: `${Math.round(Number(payload.finalScore))}%` });
  }

  let mx = padX;
  for (const item of meta) {
    ctx.fillStyle = '#7a8896';
    ctx.font = '600 24px "Segoe UI", "PT Sans", Arial, sans-serif';
    ctx.fillText(item.label, mx, y);
    ctx.fillStyle = '#4b5c6b';
    ctx.font = '400 32px "Segoe UI", "PT Sans", Arial, sans-serif';
    ctx.fillText(item.value, mx, y + 44);
    mx += Math.max(320, ctx.measureText(item.value).width + 80);
  }

  // футер
  const fy = h - 140;
  ctx.strokeStyle = '#d5dde6';
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.moveTo(padX, fy);
  ctx.lineTo(w - padX, fy);
  ctx.stroke();

  ctx.fillStyle = '#7a8896';
  ctx.font = '400 26px "Segoe UI", "PT Sans", Arial, sans-serif';
  ctx.fillText('Документ сформирован автоматически по итогам прохождения обучения.', padX, fy + 48);
  ctx.fillText('ADMSR', w - padX - ctx.measureText('ADMSR').width, fy + 48);

  return canvas;
}

function certificateStyles(): string {
  return `
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { margin: 0; background: #ffffff; }
    .cert {
      width: 1123px;
      height: 794px;
      background: #ffffff;
      border: 2px solid #1e3a5f;
      outline: 1px solid #10b981;
      outline-offset: -10px;
      padding: 48px 56px;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      font-family: "Segoe UI", "PT Sans", Arial, sans-serif;
      color: #1a2332;
    }
    .eyebrow {
      text-transform: uppercase;
      letter-spacing: 0.28em;
      font-size: 12px;
      color: #5b6b7c;
      margin: 0 0 12px;
    }
    h1 {
      margin: 0;
      font-size: 42px;
      font-weight: 600;
      letter-spacing: 0.04em;
      color: #0f2744;
    }
    .lead { margin: 28px 0 0; font-size: 18px; color: #4b5c6b; }
    .name { margin: 12px 0 0; font-size: 34px; font-weight: 600; color: #0f2744; }
    .course-label { margin: 28px 0 0; font-size: 15px; color: #5b6b7c; }
    .course {
      margin: 8px 0 0;
      font-size: 26px;
      font-weight: 600;
      color: #0b7a55;
      line-height: 1.3;
    }
    .meta {
      display: flex;
      gap: 40px;
      flex-wrap: wrap;
      margin-top: 36px;
      font-size: 14px;
      color: #4b5c6b;
    }
    .meta strong {
      display: block;
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: 0.12em;
      color: #7a8896;
      margin-bottom: 4px;
      font-weight: 600;
    }
    .footer {
      display: flex;
      justify-content: space-between;
      align-items: flex-end;
      gap: 24px;
      border-top: 1px solid #d5dde6;
      padding-top: 18px;
      margin-top: 24px;
      font-size: 12px;
      color: #7a8896;
    }
  `;
}

function buildCertificateBody(payload: CourseCertificatePayload): string {
  const title = escapeHtml(payload.courseTitle || 'Обучение');
  const fio = escapeHtml(payload.userFio || 'Сотрудник');
  const date = escapeHtml(formatDate(payload.completedAt));
  const ofo = payload.ofoName ? escapeHtml(String(payload.ofoName)) : '';
  const score =
    payload.finalScore != null && Number.isFinite(Number(payload.finalScore))
      ? `${Math.round(Number(payload.finalScore))}%`
      : '';

  return `
    <div class="cert">
      <div>
        <p class="eyebrow">Корпоративный портал ADMSR</p>
        <h1>Сертификат</h1>
        <p class="lead">Настоящим подтверждается, что</p>
        <p class="name">${fio}</p>
        <p class="course-label">успешно прошёл(а) обучение</p>
        <p class="course">«${title}»</p>
        <div class="meta">
          <div><strong>Дата завершения</strong>${date}</div>
          ${ofo ? `<div><strong>Подразделение</strong>${ofo}</div>` : ''}
          ${score ? `<div><strong>Итоговый результат</strong>${score}</div>` : ''}
        </div>
      </div>
      <div class="footer">
        <div>Документ сформирован автоматически по итогам прохождения обучения.</div>
        <div>ADMSR</div>
      </div>
    </div>
  `;
}

/** Скачать готовый PDF-файл. */
export async function downloadCourseCertificatePdf(payload: CourseCertificatePayload): Promise<void> {
  const canvas = drawCertificateCanvas(payload);
  const img = canvas.toDataURL('image/png', 1.0);
  const pdf = new jsPDF({ orientation: 'landscape', unit: 'mm', format: 'a4' });
  const pageW = pdf.internal.pageSize.getWidth();
  const pageH = pdf.internal.pageSize.getHeight();
  pdf.addImage(img, 'PNG', 0, 0, pageW, pageH, undefined, 'FAST');
  pdf.save(safeFileName(payload.courseTitle));
}

/** Диалог печати. */
export async function openCourseCertificatePrint(payload: CourseCertificatePayload): Promise<void> {
  const title = escapeHtml(payload.courseTitle || 'Обучение');
  const html = `<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="utf-8" />
  <title>Сертификат — ${title}</title>
  <style>
    @page { size: A4 landscape; margin: 12mm; }
    ${certificateStyles()}
    .sheet { display: flex; align-items: center; justify-content: center; min-height: 100vh; padding: 8px; }
    .cert { width: 100%; max-width: 1000px; height: auto; aspect-ratio: 1.414 / 1; }
    @media print { .sheet { padding: 0; min-height: auto; } }
  </style>
</head>
<body>
  <div class="sheet">${buildCertificateBody(payload)}</div>
</body>
</html>`;

  return new Promise((resolve, reject) => {
    document.getElementById('course-certificate-print-frame')?.remove();

    const iframe = document.createElement('iframe');
    iframe.id = 'course-certificate-print-frame';
    iframe.setAttribute('aria-hidden', 'true');
    iframe.style.cssText =
      'position:fixed;right:0;bottom:0;width:0;height:0;border:0;opacity:0;pointer-events:none;';

    let finished = false;

    const cleanup = () => {
      window.setTimeout(() => iframe.remove(), 1000);
    };

    const runPrint = () => {
      // srcdoc иногда даёт два load (about:blank + контент) — печатаем только один раз
      if (finished) return;
      const doc = iframe.contentDocument;
      const win = iframe.contentWindow;
      if (!doc?.querySelector('.cert') || !win) return;

      finished = true;
      try {
        win.focus();
        win.print();
        resolve();
      } catch (e: any) {
        reject(e instanceof Error ? e : new Error(String(e?.message || e)));
      } finally {
        cleanup();
      }
    };

    iframe.addEventListener('load', runPrint);
    // srcdoc до append — меньше лишних load-событий
    iframe.srcdoc = html;
    document.body.appendChild(iframe);

    // запасной путь, если load уже прошёл до подписки
    window.setTimeout(runPrint, 300);
  });
}
