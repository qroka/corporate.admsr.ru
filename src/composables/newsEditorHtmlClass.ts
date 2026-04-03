/**
 * Те же утилиты, что у слота `base` у `UEditor` (TipTap / ProseMirror) в Nuxt UI —
 * чтобы HTML новости на детальной странице выглядел как в редакторе.
 */
export const newsEditorHtmlClass = [
  'w-full outline-none *:my-5 *:first:mt-0 *:last:mb-0 sm:px-8 selection:bg-primary/20',
  '[&_p]:leading-7',
  '[&_a]:text-primary [&_a]:border-b [&_a]:border-transparent [&_a]:hover:border-primary [&_a]:font-medium [&_a]:transition-colors',
  '[&_a>code]:border-dashed [&_a:hover>code]:border-primary [&_a:hover>code]:text-primary [&_a>code]:transition-colors',
  '[&_.mention]:text-primary [&_.mention]:font-medium',
  '[&_:is(h1,h2,h3,h4,h5,h6)]:text-highlighted [&_:is(h1,h2,h3,h4,h5,h6)]:font-bold',
  '[&_h1]:text-3xl',
  '[&_h2]:text-2xl',
  '[&_h3]:text-xl',
  '[&_h4]:text-lg',
  '[&_h5]:text-base',
  '[&_h6]:text-base',
  '[&_:is(h1,h2,h3,h4,h5,h6)>code]:border-dashed [&_:is(h1,h2,h3,h4,h5,h6)>code]:font-bold',
  '[&_h2>code]:text-xl/6',
  '[&_h3>code]:text-lg/5',
  '[&_blockquote]:border-s-4 [&_blockquote]:border-accented [&_blockquote]:ps-4 [&_blockquote]:italic',
  '[&_[data-type=horizontalRule]]:my-8 [&_[data-type=horizontalRule]]:py-2',
  '[&_hr]:border-t [&_hr]:border-default',
  '[&_pre]:text-sm/6 [&_pre]:border [&_pre]:border-muted [&_pre]:bg-muted [&_pre]:rounded-md [&_pre]:px-4 [&_pre]:py-3 [&_pre]:whitespace-pre-wrap [&_pre]:break-words [&_pre]:overflow-x-auto',
  '[&_pre_code]:p-0 [&_pre_code]:text-inherit [&_pre_code]:font-inherit [&_pre_code]:rounded-none [&_pre_code]:inline [&_pre_code]:border-0 [&_pre_code]:bg-transparent',
  '[&_code]:px-1.5 [&_code]:py-0.5 [&_code]:text-sm [&_code]:font-mono [&_code]:font-medium [&_code]:rounded-md [&_code]:inline-block [&_code]:border [&_code]:border-muted [&_code]:text-highlighted [&_code]:bg-muted',
  '[&_:is(ul,ol)]:ps-6',
  '[&_ul]:list-disc [&_ul]:marker:text-(--ui-border-accented)',
  '[&_ol]:list-decimal [&_ol]:marker:text-muted',
  '[&_li]:my-1.5 [&_li]:ps-1.5',
  '[&_img]:rounded-md [&_img]:block [&_img]:max-w-full',
].join(' ');
