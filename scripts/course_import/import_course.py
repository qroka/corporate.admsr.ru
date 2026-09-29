"""Размещение курса из архива через API портала (как это сделал бы админ в интерфейсе).

python import_course.py <папка архива> <tests.json> <base_url> <login> <password>
"""
import json
import os
import sys

import requests

ROOT, TESTS, BASE, LOGIN, PASSWORD = sys.argv[1:6]
tests = json.load(open(TESTS, encoding='utf-8'))
s = requests.Session()


def call(path, payload=None, files=None, data=None):
    if files:
        r = s.post(BASE + path, files=files, data=data, timeout=300)
    else:
        r = s.post(BASE + path, json=payload or {}, timeout=60)
    try:
        j = r.json()
    except Exception:
        raise SystemExit(f'{path}: HTTP {r.status_code} {r.text[:300]}')
    if not r.ok or not j.get('success'):
        raise SystemExit(f'{path}: HTTP {r.status_code} {j}')
    return j.get('data')


auth = call('/api/auth.php', {'login': LOGIN, 'password': PASSWORD})
token = (auth or {}).get('sessionToken')
if token:
    s.headers['Authorization'] = 'Bearer ' + token

COURSE_TITLE = 'Раздел 1. Основы и безопасная работа с ИИ'
TOPICS = [
    ('00 Введение и входная диагностика', 'Введение и входная диагностика',
     ['Как проходить курс.pdf'], 'Входная диагностика'),
    ('01 Что такое генеративный ИИ', 'Что такое генеративный ИИ',
     ['Что такое генеративный ИИ.mp4', 'Глоссарий.pdf', 'Миф или факт.pdf'], 'Тест: что такое генеративный ИИ'),
    ('02 Правовые и организационные основы', 'Правовые и организационные основы',
     ['Правовые и организационные основы.pdf'], 'Тест: правовые и организационные основы'),
    ('03 Безопасная подготовка информации для работы с ИИ', 'Безопасная подготовка информации для работы с ИИ',
     ['Алгоритм действий.mp4', 'Карточки ситуаций при работе с ИИ.pdf',
      'Инструкция по подготовке и обезличиванию материалов.docx', 'Памятка по работе с ИИ.docx'],
     'Тест: безопасная подготовка информации'),
    ('04 Выбор безопасного ИИ-инструмента', 'Выбор безопасного ИИ-инструмента',
     ['Выбор безопасного ИИ-инструмента.pdf'], 'Тест: выбор безопасного ИИ-инструмента'),
]

# Подсказки к вопросам из таблиц — текст условия из документа без «бумажных» инструкций.
HINTS = {
    'Слева указаны задачи': 'Вариант должен подходить и по сервису, и по сведениям для запроса.',
    'Соотнесите понятия с определениями:': 'Выберите подходящее определение.',
    'Подберите вариант для каждой задачи:': 'Вариант должен подходить и по сервису, и по сведениям для запроса.',
    'Факт, предположение или галлюцинация': (
        'Исходные данные: в 2023 году туристический поток в Сургутском районе составил 96 тыс. посещений; '
        'в 2024 году — 120 тыс. посещений; сведения о причинах роста, новых маршрутах и составе туристов '
        'отсутствуют. Определите, чем является этот ответ ИИ.'),
    'Допустимо, сначала уточнить или нельзя': (
        'Условия: сотруднику нужно выполнить рабочую задачу с помощью внешнего ИИ-сервиса. Отдельный перечень '
        'разрешённых сервисов в организации не утверждён. Оценивайте не только сервис, но и сведения, которые '
        'планируется передать. Если допустимость сведений неясна, требуется уточнение.'),
}


def hint_for(h):
    for prefix, text in HINTS.items():
        if h.startswith(prefix):
            return text
    return h


def read_text(folder):
    with open(os.path.join(ROOT, folder, 'Сопроводительный текст.txt'), encoding='utf-8-sig') as f:
        return f.read().strip()


created = call('/api/courses_create.php', {
    'title': COURSE_TITLE,
    'category': 'Безопасность',
    'shortDescription': 'Как устроен генеративный ИИ, правовые основы его использования, безопасная '
                        'подготовка информации и выбор ИИ-инструмента. Пять тем с тестами.',
    'sequentialProgress': True,
})
course_id = created['course']['id']
version_id = created['version']['id']
call('/api/courses_update.php', {'courseId': course_id, 'versionId': version_id,
                                 'requireFinalTest': False, 'generateCertificate': False})
print('курс', course_id, 'версия', version_id)

for folder, title, files, test_title in TOPICS:
    topic = call('/api/course_topics_create.php', {
        'courseId': course_id, 'versionId': version_id, 'title': title,
        'description': read_text(folder), 'isRequired': True, 'minimumActiveSeconds': 0,
    })['topic']
    tid = topic['id']
    for name in files:
        path = os.path.join(ROOT, folder, name)
        with open(path, 'rb') as fh:
            up = call('/api/course_materials_upload.php', files={'file': (name, fh)}, data={
                'topicId': str(tid), 'title': os.path.splitext(name)[0], 'description': '',
                'type': 'file', 'isRequired': '1', 'minimumActiveSeconds': '0',
            })
        print('  материал', up['materialId'], name, up['mimeType'])

    link = call('/api/course_tests_create.php', {'topicId': tid})['link']
    form = call('/api/course_tests_get.php', {'courseTestLinkId': link['id'], 'topicId': tid})['form']
    t = tests[folder]
    qs = []
    for qi, q in enumerate(t['questions']):
        if q.get('type') in ('match', 'classify'):
            items = [{'id': f'i{qi}_{k}', 'text': text} for k, text in enumerate(q['items'])]
            opts = [{'id': f'o{qi}_{j}', 'text': text} for j, text in enumerate(q['options'])]
            qs.append({
                'id': f'q{qi}', 'title': q['title'], 'hint': hint_for(q['hint']), 'type': q['type'],
                'required': True, 'options': opts, 'items': items, 'scaleMin': 1, 'scaleMax': 5,
                'scaleMinLabel': '', 'scaleMaxLabel': '',
                'correct': {items[k]['id']: opts[j]['id'] for k, j in enumerate(q['key'])},
            })
            continue
        opts = [{'id': f'o{qi}_{oi}', 'text': o['text']} for oi, o in enumerate(q['options'])]
        correct = next(f'o{qi}_{oi}' for oi, o in enumerate(q['options']) if o['correct'])
        qs.append({
            'id': f'q{qi}', 'title': q['title'], 'hint': hint_for(q['hint']), 'type': 'single',
            'required': True, 'options': opts, 'scaleMin': 1, 'scaleMax': 5,
            'scaleMinLabel': '', 'scaleMaxLabel': '', 'correct': correct,
        })
    form['title'] = test_title
    form['description'] = ('Выберите один правильный ответ в каждом задании. Проходного балла нет: результат '
                           'покажет вашу отправную точку.' if folder.startswith('00')
                           else 'Выберите один правильный ответ в каждом задании.')
    form['questions'] = qs
    form['shuffle'] = False
    form['shuffleOptions'] = False
    saved = call('/api/course_tests_update.php', {
        'courseTestLinkId': link['id'], 'testFormId': form['id'], 'form': form, 'isRequired': True,
    })
    print('  тест', saved['link']['id'], test_title, 'вопросов:', len(saved['form']['questions']))

json.dump({'courseId': course_id, 'versionId': version_id}, open(os.path.join(os.path.dirname(TESTS), 'course.json'), 'w'))
print('ГОТОВО courseId =', course_id)
