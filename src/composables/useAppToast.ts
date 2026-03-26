import { useToast } from '@nuxt/ui/composables';

const successDefaults = {
  color: 'success' as const,
  icon: 'i-lucide-circle-check',
};

/**
 * Единая точка для уведомлений приложения (обёртка над useToast из Nuxt UI).
 */
export function useAppToast() {
  const toast = useToast();

  function profileSaved() {
    toast.add({
      title: 'Профиль сохранён',
      description: 'Изменения записаны локально (демо).',
      ...successDefaults,
    });
  }

  /** После сохранения карточки пользователя в админке (демо). */
  function adminUserSaved(fullName: string) {
    toast.add({
      title: 'Пользователь сохранён',
      description: `Данные «${fullName}» обновлены (демо).`,
      ...successDefaults,
    });
  }

  /** После сохранения ОФО в админке (демо). */
  function adminOfoSaved(title: string) {
    toast.add({
      title: 'ОФО сохранено',
      description: `«${title}» — изменения записаны локально (демо).`,
      ...successDefaults,
    });
  }

  return {
    toast,
    profileSaved,
    adminUserSaved,
    adminOfoSaved,
  };
}
