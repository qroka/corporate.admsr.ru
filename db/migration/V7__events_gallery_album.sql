-- Связь мероприятия с альбомом фотогалереи
ALTER TABLE events
  ADD COLUMN IF NOT EXISTS album_id INTEGER NULL
  REFERENCES public.gallery(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_events_album_id ON events(album_id) WHERE album_id IS NOT NULL;
