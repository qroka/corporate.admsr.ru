-- V6__portal_group_course_categories.sql
-- Категории курсов, доступные группе с правом section_key = courses.
-- Значения category_key совпадают с course_courses.category (русские названия).
-- PostgreSQL. Schema: public. Идемпотентно.

CREATE TABLE IF NOT EXISTS public.portal_group_course_categories (
  group_id      bigint NOT NULL REFERENCES public.portal_access_groups(id) ON DELETE CASCADE,
  category_key  text NOT NULL,
  PRIMARY KEY (group_id, category_key)
);

CREATE INDEX IF NOT EXISTS portal_group_course_categories_cat_idx
  ON public.portal_group_course_categories(category_key);
