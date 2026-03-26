<template>
  <UMain class="flex flex-1 min-h-0 h-full w-full mx-0">
    <UContainer class="flex flex-col gap-4 w-full min-h-0 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
      <UPageHeader title="" class="border-none p-0 w-full">
        <template #title>
          <h1 class="text-4xl font-normal font-unbounded">Новичкам</h1>
        </template>
      </UPageHeader>

      <UContainer class="flex-1 min-h-0 grid place-items-center p-0">
        <UCard class="rounded-3xl w-full max-w-3xl">
          <UContainer class="flex flex-col items-center text-center gap-4 p-0 py-8 sm:py-10">
            <div class="shrink-0">
              <div class="size-20 rounded-full bg-primary/10 ring-1 ring-primary/20 grid place-items-center">
                <UIcon name="i-lucide-construction" class="size-12 text-primary" />
              </div>
            </div>
            <div class="min-w-0">
              <div class="text-2xl sm:text-3xl font-semibold text-highlighted">Раздел в разработке</div>
              <div class="mt-2 text-base sm:text-lg text-toned">
                Скоро здесь появятся инструкции, контакты и полезные ссылки для новых сотрудников.
              </div>
            </div>
          </UContainer>
        </UCard>
      </UContainer>
    </UContainer>
  </UMain>
</template>

<script setup>
</script>

