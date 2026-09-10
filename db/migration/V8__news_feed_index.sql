-- Index for news feed cursor pagination: ORDER BY date DESC, id DESC
CREATE INDEX IF NOT EXISTS idx_news_date_id_desc
  ON public.news (date DESC, id DESC);
