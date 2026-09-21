-- Certificate generation flag for course versions
ALTER TABLE public.course_versions
  ADD COLUMN IF NOT EXISTS generate_certificate boolean NOT NULL DEFAULT false;
