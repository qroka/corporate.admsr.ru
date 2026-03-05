<template>
  <div class="app-root h-screen overflow-hidden">
    <UApp :locale="ru">
      <!-- Основной лейаут приложения: вертикальный флекс-контейнер на высоту экрана -->
      <div class="h-screen overflow-hidden flex flex-col justify-start p-6 gap-6 transition-colors"
        :class="containerClass">
        <!-- Шапка портала: логотип, основная навигация и блок профиля пользователя -->
        <header class="flex flex-row h-fit justify-between w-full">
          <RouterLink :to="{ name: 'home' }"
            class="group w-[72px] sm:w-auto bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center sm:justify-start gap-1.5 p-3 sm:pr-5 rounded-full cursor-pointer transition-all duration-300 ease-out">
            <!-- Блок логотипа и названия портала -->
            <div class="header-logo flex items-center gap-1.5">
              <div class="relative w-12 h-12">
                <!-- Базовый логотип: всегда виден, в тёмной теме прячется при hover -->
                <svg :class="[
                  'absolute w-12 h-12 duration-300 ease-out',
                  isDark
                    ? 'opacity-100 dark:fill-zinc-500 group-hover:opacity-0'
                    : 'opacity-100 fill-zinc-400 group-hover:fill-brand'
                ]" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path
                    d="M37.6895 8C38.963 8.00009 39.9969 9.0301 39.9971 10.2988V11.2412L40 24.0781C40 32.8696 32.8431 40 24.0186 40H8L35.2393 12.8623L30.3594 8H37.6895ZM24.0586 23.9971L8.00195 39.9971V34.5879H12.252V30.3545H8.00195V26.1172H12.252V21.8799H8.00195V17.6426H12.252V13.4092H8.00195V8L24.0586 23.9971ZM12.2578 26.1143V30.3506H16.5107V26.1143H12.2578ZM16.5176 21.8799V26.1172H20.7705V21.8799H16.5176ZM12.2578 17.6426V21.8799H16.5107V17.6426H12.2578Z" />

                </svg>
                <!-- Логотип с подсветкой: показывается только в тёмной теме при hover -->
                <svg :class="[
                  'overflow-visible absolute w-12 h-12 transition-opacity duration-300 ease-out',
                  isDark ? 'opacity-0 group-hover:opacity-100' : 'opacity-0'
                ]" viewBox="16 16 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <g filter="url(#filter0_dddddd_1609_3992)">
                    <path
                      d="M53.8815 24.192C55.155 24.1921 56.1889 25.2221 56.1891 26.4908V27.4332L56.192 40.2701C56.192 49.0616 49.0351 56.192 40.2106 56.192H24.192L51.4313 29.0543L46.5514 24.192H53.8815ZM40.2506 40.1891L24.194 56.1891V50.7799H28.444V46.5465H24.194V42.3092H28.444V38.0719H24.194V33.8346H28.444V29.6012H24.194V24.192L40.2506 40.1891ZM28.4498 42.3062V46.5426H32.7028V42.3062H28.4498ZM32.7096 38.0719V42.3092H36.9625V38.0719H32.7096ZM28.4498 33.8346V38.0719H32.7028V33.8346H28.4498Z"
                      fill="#50C891" />
                  </g>
                  <defs>
                    <filter id="filter0_dddddd_1609_3992" x="-7.99998" y="-8.00001" width="96.384" height="96.384"
                      filterUnits="userSpaceOnUse" color-interpolation-filters="sRGB">
                      <feFlood flood-opacity="0" result="BackgroundImageFix" />
                      <feColorMatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
                        result="hardAlpha" />
                      <feOffset />
                      <feGaussianBlur stdDeviation="0.288" />
                      <feColorMatrix type="matrix"
                        values="0 0 0 0 0.313726 0 0 0 0 0.784314 0 0 0 0 0.568627 0 0 0 1 0" />
                      <feBlend mode="normal" in2="BackgroundImageFix" result="effect1_dropShadow_1609_3992" />
                      <feColorMatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
                        result="hardAlpha" />
                      <feOffset />
                      <feGaussianBlur stdDeviation="0.576" />
                      <feColorMatrix type="matrix"
                        values="0 0 0 0 0.313726 0 0 0 0 0.784314 0 0 0 0 0.568627 0 0 0 1 0" />
                      <feBlend mode="normal" in2="effect1_dropShadow_1609_3992" result="effect2_dropShadow_1609_3992" />
                      <feColorMatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
                        result="hardAlpha" />
                      <feOffset />
                      <feGaussianBlur stdDeviation="2.016" />
                      <feColorMatrix type="matrix"
                        values="0 0 0 0 0.313726 0 0 0 0 0.784314 0 0 0 0 0.568627 0 0 0 1 0" />
                      <feBlend mode="normal" in2="effect2_dropShadow_1609_3992" result="effect3_dropShadow_1609_3992" />
                      <feColorMatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
                        result="hardAlpha" />
                      <feOffset />
                      <feGaussianBlur stdDeviation="4.032" />
                      <feColorMatrix type="matrix"
                        values="0 0 0 0 0.313726 0 0 0 0 0.784314 0 0 0 0 0.568627 0 0 0 1 0" />
                      <feBlend mode="normal" in2="effect3_dropShadow_1609_3992" result="effect4_dropShadow_1609_3992" />
                      <feColorMatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
                        result="hardAlpha" />
                      <feOffset />
                      <feGaussianBlur stdDeviation="6.912" />
                      <feColorMatrix type="matrix"
                        values="0 0 0 0 0.313726 0 0 0 0 0.784314 0 0 0 0 0.568627 0 0 0 1 0" />
                      <feBlend mode="normal" in2="effect4_dropShadow_1609_3992" result="effect5_dropShadow_1609_3992" />
                      <feColorMatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
                        result="hardAlpha" />
                      <feOffset />
                      <feGaussianBlur stdDeviation="12.096" />
                      <feColorMatrix type="matrix"
                        values="0 0 0 0 0.313726 0 0 0 0 0.784314 0 0 0 0 0.568627 0 0 0 1 0" />
                      <feBlend mode="normal" in2="effect5_dropShadow_1609_3992" result="effect6_dropShadow_1609_3992" />
                      <feBlend mode="normal" in="SourceGraphic" in2="effect6_dropShadow_1609_3992" result="shape" />
                    </filter>
                  </defs>
                </svg>
              </div>


              <span
                class="hidden sm:inline text-xl font-display font-normal leading-12 text-zinc-700 dark:text-zinc-50">
                Корпоративный портал
              </span>
            </div>
          </RouterLink>
          <!-- Основная навигация по разделам портала (скрыта ниже xl, в бургер-меню) -->
          <nav
            class="header-nav hidden lg:flex bg-zinc-100 dark:bg-zinc-800 rounded-full w-fit items-center gap-2 p-3 relative z-0">
            <RouterLink :to="{ name: 'events' }"
              class="relative z-0 flex items-center gap-[10px] rounded-full w-12 h-12 min-w-12 max-w-12 px-0 justify-center 3xl:w-auto 3xl:h-auto 3xl:min-w-0 3xl:max-w-none 3xl:px-3 3xl:justify-start 3xl:shrink-0 cursor-pointer transition-all duration-300 ease-out group"
              :class="activeNav === 'events'
                ? 'bg-brand text-zinc-50 z-10 shadow-none dark:text-zinc-50 dark:shadow-brand'
                : 'bg-zinc-200 text-zinc-400 hover:bg-zinc-900 dark:bg-zinc-700 dark:text-zinc-400'">
              <CalendarIcon class="w-6 h-6" />
              <span class="leading-12 text-base transition-colors duration-200 hidden 3xl:inline" :class="activeNav === 'events'
                ? 'text-zinc-50'
                : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                Мероприятия
              </span>
            </RouterLink>
            <RouterLink :to="{ name: 'gallery' }"
              class="relative z-0 flex items-center gap-[10px] rounded-full w-12 h-12 min-w-12 max-w-12 px-0 justify-center 3xl:w-auto 3xl:h-auto 3xl:min-w-0 3xl:max-w-none 3xl:px-3 3xl:justify-start 3xl:shrink-0 cursor-pointer transition-all duration-300 ease-out group"
              :class="activeNav === 'gallery'
                ? 'bg-brand text-zinc-50 z-10 shadow-none dark:text-zinc-50 dark:shadow-brand'
                : 'bg-zinc-200 text-zinc-400 hover:bg-zinc-900 dark:bg-zinc-700 dark:text-zinc-400'">
              <PhotoIcon class="w-6 h-6" />
              <span class="leading-12 text-base transition-colors duration-200 hidden 3xl:inline" :class="activeNav === 'gallery'
                ? 'text-zinc-50'
                : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                Фотогалерея
              </span>
            </RouterLink>
            <RouterLink :to="{ name: 'newcomers' }"
              class="relative z-0 flex items-center gap-[10px] rounded-full w-12 h-12 min-w-12 max-w-12 px-0 justify-center 3xl:w-auto 3xl:h-auto 3xl:min-w-0 3xl:max-w-none 3xl:px-3 3xl:justify-start 3xl:shrink-0 cursor-pointer transition-all duration-300 ease-out group"
              :class="activeNav === 'newcomers'
                ? 'bg-brand text-zinc-50 z-10 shadow-none dark:text-zinc-50 dark:shadow-brand'
                : 'bg-zinc-200 text-zinc-400 hover:bg-zinc-900 dark:bg-zinc-700 dark:text-zinc-400'">
              <SparklesIcon class="w-6 h-6" />
              <span class="leading-12 text-base transition-colors duration-200 hidden 3xl:inline" :class="activeNav === 'newcomers'
                ? 'text-zinc-50'
                : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                Новичкам
              </span>
            </RouterLink>
            <RouterLink :to="{ name: 'culture' }"
              class="relative z-0 flex items-center gap-[10px] rounded-full w-12 h-12 min-w-12 max-w-12 px-0 justify-center 3xl:w-auto 3xl:h-auto 3xl:min-w-0 3xl:max-w-none 3xl:px-3 3xl:justify-start 3xl:shrink-0 cursor-pointer transition-all duration-300 ease-out group"
              :class="activeNav === 'culture'
                ? 'bg-brand text-zinc-50 z-10 shadow-none dark:text-zinc-50 dark:shadow-brand'
                : 'bg-zinc-200 text-zinc-400 hover:bg-zinc-900 dark:bg-zinc-700 dark:text-zinc-400'">
              <UsersIcon class="w-6 h-6" />
              <span class="leading-12 text-base transition-colors duration-200 hidden 3xl:inline" :class="activeNav === 'culture'
                ? 'text-zinc-50'
                : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                Корпоративная культура
              </span>
            </RouterLink>
          </nav>
          <!-- Правая часть шапки: иконки уведомлений/поиска и профиль (скрыта ниже xl) -->
          <div class="header-right hidden xl:flex">
            <!-- Блок уведомлений и поиска -->
            <div class="notification-search bg-zinc-100 dark:bg-zinc-800 rounded-full w-fit flex items-center p-3">
              <button type="button"
                class="btn-icon bg-zinc-200 text-zinc-400 dark:bg-zinc-700 dark:text-zinc-400 flex rounded-full h-12 items-center px-3 gap-[10px] cursor-pointer hover:bg-zinc-300 dark:hover:bg-zinc-600 transition-all duration-300 ease-out">
                <BellIcon class="w-6 h-6" />
              </button>
              <button type="button"
                class="btn-icon bg-zinc-200 text-zinc-400 dark:bg-zinc-700 dark:text-zinc-400 flex rounded-full h-12 items-center px-3 gap-[10px] cursor-pointer hover:bg-zinc-300 dark:hover:bg-zinc-600 transition-all duration-300 ease-out">
                <MagnifyingGlassIcon class="w-6 h-6 " />
              </button>
            </div>
            <RouterLink :to="{ name: 'profile' }"
              class="bg-zinc-100 dark:bg-zinc-800 rounded-full w-fit cursor-pointer flex items-center p-3 hover:bg-zinc-200 dark:hover:bg-zinc-700 group transition-all duration-300 ease-out">
              <!-- Аватар и краткая информация о пользователе -->
              <div
                class="profile-icon bg-zinc-300 dark:bg-zinc-700 flex rounded-full h-12 w-12 items-center px-2 gap-[10px] group-hover:bg-zinc-100 dark:group-hover:bg-zinc-800 transition-all duration-300 ease-out">
                <img src="/src/img/sticker 1.png" />
              </div>
              <div class="profile-meta flex flex-col px-3">
                <div class="profile-name leading-6 text-base text-zinc-700 dark:text-zinc-50">
                  Константин Константинович
                </div>
                <div class="profile-position leading-4 text-sm text-zinc-400 dark:text-zinc-500">
                  Инженер
                </div>
              </div>
            </RouterLink>
          </div>
          <!-- Кнопка бургер-меню (только ниже xl) -->
          <div class="xl:hidden bg-zinc-100 dark:bg-zinc-800 rounded-full w-fit flex items-center p-3">
            <UButton color="neutral" variant="ghost"
              class="xl:hidden bg-zinc-200 text-zinc-400 dark:bg-zinc-700 dark:text-zinc-400 flex rounded-full h-12 items-center px-3 gap-[10px] cursor-pointer hover:bg-zinc-300 dark:hover:bg-zinc-600 transition-all duration-300 ease-out"
              aria-label="Открыть меню" @click="menuOpen = true">
              <Bars3Icon class="w-6 h-6" />
            </UButton>
          </div>
        </header>

        <!-- Slideover: навигация, профиль и боковая панель для экранов < xl -->
        <USlideover v-model:open="menuOpen" side="right" title="Меню" class="xl:hidden">
          <template #body>
            <div class="flex flex-col gap-6 py-2">
              <!-- Аккаунт (сверху): уведомления, поиск, профиль -->
              <div class="flex flex-col gap-1">
                <p class="text-xs font-medium text-zinc-500 dark:text-zinc-400 px-3 mb-1">Аккаунт</p>
                <RouterLink :to="{ name: 'profile' }"
                  class="flex items-center gap-3 rounded-full px-4 py-3 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                  @click="menuOpen = false">
                  <div class="w-12 h-12 rounded-full bg-zinc-300 dark:bg-zinc-700 overflow-hidden shrink-0">
                    <img src="/src/img/sticker 1.png" alt="" class="w-full h-full object-cover" />
                  </div>
                  <div class="text-left">
                    <div class="font-medium text-zinc-900 dark:text-zinc-50">Константин Константинович</div>
                    <div class="text-sm text-zinc-500 dark:text-zinc-400">Инженер</div>
                  </div>
                </RouterLink>
                <button type="button"
                  class="flex items-center gap-3 rounded-full px-4 py-3 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors w-full text-left">
                  <MagnifyingGlassIcon class="w-5 h-5 shrink-0" />
                  <span>Поиск</span>
                </button>
                <button type="button"
                  class="flex items-center gap-3 rounded-full px-4 py-3 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors w-full text-left">
                  <BellIcon class="w-5 h-5 shrink-0" />
                  <span>Уведомления</span>
                </button>
              </div>

              <!-- Навигация по разделам -->
              <div class="flex flex-col gap-1">
                <p class="text-xs font-medium text-zinc-500 dark:text-zinc-400 px-3 mb-1">Разделы</p>
                <RouterLink :to="{ name: 'events' }"
                  class="flex items-center gap-3 rounded-full px-4 py-3 transition-colors"
                  :class="activeNav === 'events' ? 'bg-brand text-white' : 'hover:bg-zinc-100 dark:hover:bg-zinc-800'"
                  @click="menuOpen = false">
                  <CalendarIcon class="w-5 h-5 shrink-0" />
                  <span>Мероприятия</span>
                </RouterLink>
                <RouterLink :to="{ name: 'gallery' }"
                  class="flex items-center gap-3 rounded-full px-4 py-3 transition-colors"
                  :class="activeNav === 'gallery' ? 'bg-brand text-white' : 'hover:bg-zinc-100 dark:hover:bg-zinc-800'"
                  @click="menuOpen = false">
                  <PhotoIcon class="w-5 h-5 shrink-0" />
                  <span>Фотогалерея</span>
                </RouterLink>
                <RouterLink :to="{ name: 'newcomers' }"
                  class="flex items-center gap-3 rounded-full px-4 py-3 transition-colors"
                  :class="activeNav === 'newcomers' ? 'bg-brand text-white' : 'hover:bg-zinc-100 dark:hover:bg-zinc-800'"
                  @click="menuOpen = false">
                  <SparklesIcon class="w-5 h-5 shrink-0" />
                  <span>Новичкам</span>
                </RouterLink>
                <RouterLink :to="{ name: 'culture' }"
                  class="flex items-center gap-3 rounded-full px-4 py-3 transition-colors"
                  :class="activeNav === 'culture' ? 'bg-brand text-white' : 'hover:bg-zinc-100 dark:hover:bg-zinc-800'"
                  @click="menuOpen = false">
                  <UsersIcon class="w-5 h-5 shrink-0" />
                  <span>Корпоративная культура</span>
                </RouterLink>
              </div>

              <!-- Быстрые ссылки (сервисы) -->
              <div class="flex flex-col gap-1">
                <p class="text-xs font-medium text-zinc-500 dark:text-zinc-400 px-3 mb-1">Сервисы</p>
                <RouterLink :to="{ name: 'absence-journal' }"
                  class="flex items-center gap-3 rounded-full px-4 py-2.5 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                  @click="menuOpen = false">
                  <CalendarDaysIcon class="w-5 h-5 shrink-0" />
                  <span>Журнал отсутствия</span>
                </RouterLink>
                <RouterLink :to="{ name: 'applications' }"
                  class="flex items-center gap-3 rounded-full px-4 py-2.5 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                  @click="menuOpen = false">
                  <BookOpenIcon class="w-5 h-5 shrink-0" />
                  <span>Заявки</span>
                </RouterLink>
                <RouterLink :to="{ name: 'knowledge-base' }"
                  class="flex items-center gap-3 rounded-full px-4 py-2.5 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                  @click="menuOpen = false">
                  <BookOpenIcon class="w-5 h-5 shrink-0" />
                  <span>База знаний</span>
                </RouterLink>
                <RouterLink :to="{ name: 'personnel-reserve' }"
                  class="flex items-center gap-3 rounded-full px-4 py-2.5 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                  @click="menuOpen = false">
                  <UserGroupIcon class="w-5 h-5 shrink-0" />
                  <span>Кадровый резерв</span>
                </RouterLink>
                <RouterLink :to="{ name: 'tests' }"
                  class="flex items-center gap-3 rounded-full px-4 py-2.5 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                  @click="menuOpen = false">
                  <ClipboardDocumentCheckIcon class="w-5 h-5 shrink-0" />
                  <span>Тесты</span>
                </RouterLink>
              </div>

              <!-- Подразделения -->
              <div class="flex flex-col gap-1">
                <p class="text-xs font-medium text-zinc-500 dark:text-zinc-400 px-3 mb-1">Управление муниципальной
                  службы, кадров и наград</p>
                <RouterLink :to="{ name: 'hr-department' }"
                  class="flex items-center gap-3 rounded-full px-4 py-2.5 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                  @click="menuOpen = false">
                  <BuildingOffice2Icon class="w-5 h-5 shrink-0" />
                  <span>Отдел кадров</span>
                </RouterLink>
                <RouterLink :to="{ name: 'municipal-service' }"
                  class="flex items-center gap-3 rounded-full px-4 py-2.5 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                  @click="menuOpen = false">
                  <BuildingOfficeIcon class="w-5 h-5 shrink-0" />
                  <span>Отдел муниципальной службы</span>
                </RouterLink>
                <RouterLink :to="{ name: 'development-motivation' }"
                  class="flex items-center gap-3 rounded-full px-4 py-2.5 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                  @click="menuOpen = false">
                  <SparklesIcon class="w-5 h-5 shrink-0" />
                  <span>Отдел развития и мотивации персонала</span>
                </RouterLink>
              </div>

              <!-- Служебные действия -->
              <div class="flex flex-col gap-1 pt-2 border-t border-zinc-200 dark:border-zinc-700">
                <button type="button"
                  class="flex items-center gap-3 rounded-full px-4 py-2.5 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors w-full text-left"
                  @click="startThemeTransition">
                  <span class="relative w-5 h-5 shrink-0">
                    <MoonIcon class="absolute inset-0 w-5 h-5" :class="isDark ? 'opacity-0' : 'opacity-100'" />
                    <SunIcon class="absolute inset-0 w-5 h-5" :class="isDark ? 'opacity-100' : 'opacity-0'" />
                  </span>
                  <span>{{ isDark ? 'Светлая тема' : 'Тёмная тема' }}</span>
                </button>
                <button type="button"
                  class="flex items-center gap-3 rounded-full px-4 py-2.5 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors w-full text-left">
                  <QuestionMarkCircleIcon class="w-5 h-5 shrink-0" />
                  <span>Справка</span>
                </button>
                <button type="button"
                  class="flex items-center gap-3 rounded-full px-4 py-2.5 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors w-full text-left">
                  <ArrowRightOnRectangleIcon class="w-5 h-5 shrink-0" />
                  <span>Выход из личного кабинета</span>
                </button>
              </div>
            </div>
          </template>
        </USlideover>

        <!-- Основной контент: слева вертикальное меню (только xl+), справа область под страницы -->
        <main class="flex flex-1 w-full h-full gap-6 min-h-0">
          <!-- Левая колонка с быстрыми действиями (скрыта ниже xl, дублируется в бургер-меню) -->
          <aside class="hidden xl:flex flex-col justify-between shrink-0 z-20">
            <!-- Верхняя группа: функциональные сервисы для сотрудников -->
            <aside class="flex flex-col">
              <!-- Блок быстрых ссылок на основные сервисы -->
              <aside class="bg-zinc-100 dark:bg-zinc-800 rounded-full w-18 flex flex-col items-center p-3">
                <RouterLink :to="{ name: 'absence-journal' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'absence-journal'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <CalendarDaysIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'absence-journal'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Журнал отсутствия
                  </span>
                </RouterLink>
                <RouterLink :to="{ name: 'applications' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'applications'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <BookOpenIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'applications'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Заявки
                  </span>
                </RouterLink>
                <RouterLink :to="{ name: 'knowledge-base' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'knowledge-base'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <BookOpenIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'knowledge-base'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    База знаний
                  </span>
                </RouterLink>
                <RouterLink :to="{ name: 'personnel-reserve' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'personnel-reserve'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <UserGroupIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'personnel-reserve'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Кадровый резерв
                  </span>
                </RouterLink>
                <RouterLink :to="{ name: 'tests' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'tests'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <ClipboardDocumentCheckIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'tests'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Тесты
                  </span>
                </RouterLink>
              </aside>


              <!-- Блок ссылок на подразделения HR -->
              <aside class="bg-zinc-100 dark:bg-zinc-800 rounded-full w-18 flex flex-col items-center p-3">
                <RouterLink :to="{ name: 'hr-department' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'hr-department'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <BuildingOffice2Icon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'hr-department'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Отдел кадров
                  </span>
                </RouterLink>
                <RouterLink :to="{ name: 'municipal-service' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'municipal-service'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <BuildingOfficeIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'municipal-service'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Отдел муниципальной службы
                  </span>
                </RouterLink>
                <RouterLink :to="{ name: 'development-motivation' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'development-motivation'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <SparklesIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'development-motivation'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Отдел развития и мотивации персонала
                  </span>
                </RouterLink>
              </aside>
            </aside>

            <!-- Нижний блок: служебные действия (тема, помощь, выход) -->
            <aside class="bg-zinc-100 dark:bg-zinc-800 rounded-full w-18 flex flex-col items-center p-3">
              <!-- Переключатель светлой / тёмной темы -->
              <button type="button"
                class="btn-icon group self-start z-10 bg-zinc-200 dark:bg-zinc-700 text-zinc-400 flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out hover:bg-zinc-900"
                @click="startThemeTransition">
                <span class="relative w-6 h-6 shrink-0">
                  <!-- Луна -->
                  <MoonIcon class="absolute inset-0 w-6 h-6 transition-all duration-300 ease-out"
                    :class="isDark ? 'opacity-0 -rotate-90 scale-75' : 'opacity-100 rotate-0 scale-100'" />
                  <!-- Солнце -->
                  <SunIcon class="absolute inset-0 w-6 h-6 transition-all duration-300 ease-out"
                    :class="isDark ? 'opacity-100 rotate-0 scale-100' : 'opacity-0 rotate-90 scale-75'" />
                </span>
                <span
                  class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                  :class="isDark ? 'text-zinc-50' : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                  {{ isDark ? 'Светлая тема' : 'Тёмная тема' }}
                </span>
              </button>
              <!-- Справка -->
              <button type="button"
                class="btn-icon group self-start z-10 bg-zinc-200 dark:bg-zinc-700 text-zinc-400 flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out hover:bg-zinc-900">
                <QuestionMarkCircleIcon class="w-6 h-6 shrink-0" />
                <span
                  class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                  :class="'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                  Справка
                </span>
              </button>
              <!-- Выход -->
              <button type="button"
                class="btn-icon group self-start z-10 bg-zinc-200 dark:bg-zinc-700 text-zinc-400 flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out hover:bg-zinc-900">
                <ArrowRightOnRectangleIcon class="w-6 h-6 shrink-0" />
                <span
                  class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                  :class="'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                  Выход из личного кабинета
                </span>
              </button>
            </aside>
          </aside>

          <!-- Область для контента страниц, подключённых через Vue Router -->
          <section class="flex-1 min-w-0 min-h-0">
            <RouterView />
          </section>
        </main>
      </div>
    </UApp>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { ru } from '@nuxt/ui/locale';
import { useRoute } from 'vue-router';
import {
  ArrowRightOnRectangleIcon,
  Bars3Icon,
  BellIcon,
  BookOpenIcon,
  BuildingOffice2Icon,
  BuildingOfficeIcon,
  CalendarDaysIcon,
  CalendarIcon,
  ClipboardDocumentCheckIcon,
  MagnifyingGlassIcon,
  MoonIcon,
  PhotoIcon,
  QuestionMarkCircleIcon,
  SparklesIcon,
  SunIcon,
  UserGroupIcon,
  UsersIcon,
} from '@heroicons/vue/24/outline';

// Открытие бургер-меню (Slideover) на экранах < xl
const menuOpen = ref(false);

// Текущий маршрут (используется для подсветки активных пунктов навигации)
const route = useRoute();

// Имя активного пункта верхнего меню (events / gallery / newcomers / culture)
const activeNav = computed(() => (route.name ?? 'events'));

// Закрывать бургер-меню при переходе на другую страницу
watch(() => route.fullPath, () => {
  menuOpen.value = false;
});

// --- Поддержка светлой / тёмной темы для Nuxt UI ---
const COLOR_MODE_KEY = 'ui-color-mode';
const isDark = ref(false);

const applyColorMode = (mode) => {
  const root = document.documentElement;
  if (!root) return;

  if (mode === 'dark') {
    root.classList.add('dark');
  } else {
    root.classList.remove('dark');
  }
};

onMounted(() => {
  const saved = localStorage.getItem(COLOR_MODE_KEY);
  let mode;

  if (saved === 'light' || saved === 'dark') {
    mode = saved;
  } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    mode = 'dark';
  } else {
    mode = 'light';
  }

  isDark.value = mode === 'dark';
  applyColorMode(mode);
});

watch(isDark, (value) => {
  const mode = value ? 'dark' : 'light';
  localStorage.setItem(COLOR_MODE_KEY, mode);
  applyColorMode(mode);
});

const toggleColorMode = () => {
  isDark.value = !isDark.value;
};

// Анимированная смена темы с помощью View Transitions API
const startThemeTransition = (event) => {
  const anyDoc = document;

  if (!anyDoc.startViewTransition) {
    toggleColorMode();
    return;
  }

  const x = event.clientX;
  const y = event.clientY;
  const endRadius =
    Math.hypot(
      Math.max(x, window.innerWidth - x),
      Math.max(y, window.innerHeight - y),
    ) + 8;

  const transition = anyDoc.startViewTransition(() => {
    toggleColorMode();
  });

  transition.ready.then(() => {
    const duration = 600;
    anyDoc.documentElement.animate(
      {
        clipPath: [
          `circle(0px at ${x}px ${y}px)`,
          `circle(${endRadius}px at ${x}px ${y}px)`,
        ],
      },
      {
        duration,
        easing: 'cubic-bezier(.76,.32,.29,.99)',
        pseudoElement: '::view-transition-new(root)',
      },
    );
  });
};

// Класс фона/текста для корневого контейнера (визуальное переключение темы)
const containerClass = computed(() =>
  isDark.value ? 'bg-zinc-950 text-zinc-50' : 'bg-white text-zinc-900',
);
</script>

<style>
::view-transition-old(root),
::view-transition-new(root) {
  animation: none;
  mix-blend-mode: normal;
}

::view-transition-new(root) {
  z-index: 9999;
}

::view-transition-old(root) {
  z-index: 1;
}
</style>
