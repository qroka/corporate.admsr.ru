import { type Question } from './questionTypes';

export type TestKind = 'test' | 'survey' | 'poll';
export type Visibility = 'public' | 'private';
export type ShowResult = 'immediate' | 'after' | 'never';

export type TestForm = {
  id: number | null;
  title: string;
  description: string;
  kind: TestKind;
  visibility: Visibility;
  recipients: number[];
  // Параметры
  shuffle: boolean;
  shuffleOptions: boolean;
  showProgress: boolean;
  freeNavigation: boolean;
  anonymous: boolean;
  usePassingScore: boolean;
  passingScore: number;
  showCorrectAnswers: boolean;
  allowChangeAnswer: boolean;
  liveResults: boolean;
  allowRevote: boolean;
  completionMessage: string;
  notifyAdmin: boolean;
  // Доступ
  restrictByOfo: boolean;
  ofoIds: number[];
  questions: Question[];
  // Ограничения
  useTimeLimit: boolean;
  timeLimit: string;
  limitAttempts: boolean;
  attempts: number;
  useStart: boolean;
  startsAt: string;
  useEnd: boolean;
  endsAt: string;
  showResult: ShowResult;
  accessByLink: boolean;
  // Мета
  createdAt?: string;
  updatedAt?: string;
};

export function createEmptyForm(): TestForm {
  const todayISO = new Date().toISOString().slice(0, 10);
  return {
    id: null,
    title: '',
    description: '',
    kind: 'test',
    visibility: 'public',
    recipients: [],
    shuffle: false,
    shuffleOptions: false,
    showProgress: false,
    freeNavigation: false,
    anonymous: false,
    usePassingScore: false,
    passingScore: 70,
    showCorrectAnswers: false,
    allowChangeAnswer: false,
    liveResults: false,
    allowRevote: false,
    completionMessage: '',
    notifyAdmin: false,
    restrictByOfo: false,
    ofoIds: [],
    questions: [],
    useTimeLimit: false,
    timeLimit: '',
    limitAttempts: false,
    attempts: 1,
    useStart: true,
    startsAt: todayISO,
    useEnd: false,
    endsAt: '',
    showResult: 'after',
    accessByLink: false,
  };
}

export function cloneForm(f: TestForm): TestForm {
  return JSON.parse(JSON.stringify(f));
}
