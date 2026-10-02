import { useToast } from '@nuxt/ui/composables';

const successDefaults = {
  color: 'success' as const,
  icon: 'i-lucide-circle-check',
};

const errorDefaults = {
  color: 'error' as const,
  icon: 'i-lucide-alert-circle',
};

/**
 * Единая точка для уведомлений приложения (обёртка над useToast из Nuxt UI).
 */
export function useAppToast() {
  const toast = useToast();

  function success(title: string, description?: string) {
    toast.add({ title, description, ...successDefaults });
  }

  function error(title: string, description?: string) {
    toast.add({ title, description, ...errorDefaults });
  }

  /** Вызывается после успешного ответа сервера — изменения реально записаны. */
  function profileSaved() {
    toast.add({ title: 'Профиль сохранён', ...successDefaults });
  }

  /** Вызывается после успешного PUT /api/users.php — изменения реально записаны. */
  function adminUserSaved(fullName: string) {
    toast.add({
      title: 'Пользователь сохранён',
      description: `Данные «${fullName}» обновлены.`,
      ...successDefaults,
    });
  }

  return {
    toast,
    success,
    error,
    profileSaved,
    adminUserSaved,
  };
}
