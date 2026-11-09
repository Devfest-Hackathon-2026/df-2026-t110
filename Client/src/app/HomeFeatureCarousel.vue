<script setup lang="ts">
import { ref } from 'vue'
import { Icon } from '@iconify/vue'
import arrowLeftIcon from '@iconify-icons/lucide/arrow-left'
import arrowRightIcon from '@iconify-icons/lucide/arrow-right'

interface Slide {
  image: string
  alt: string
  caption: string
}

const props = defineProps<{ label: string; slides: Slide[] }>()
const active = ref(0)
let pointerStartX: number | null = null

function goTo(index: number) {
  active.value = (index + props.slides.length) % props.slides.length
}

function onPointerDown(event: PointerEvent) {
  if (event.pointerType === 'mouse') return
  pointerStartX = event.clientX
}

function onPointerUp(event: PointerEvent) {
  if (pointerStartX === null) return
  const distance = event.clientX - pointerStartX
  pointerStartX = null
  if (Math.abs(distance) > 48) goTo(active.value + (distance < 0 ? 1 : -1))
}
</script>

<template>
  <div class="home-carousel" role="region" aria-roledescription="スライドショー" :aria-label="label">
    <div
      class="home-carousel-view"
      @pointerdown="onPointerDown"
      @pointerup="onPointerUp"
      @pointercancel="pointerStartX = null"
    >
      <Transition name="home-carousel-slide" mode="out-in">
        <figure :key="active" class="home-carousel-figure">
          <img :src="slides[active]?.image" :alt="slides[active]?.alt" width="1200" height="640" />
          <figcaption aria-live="polite">{{ slides[active]?.caption }}</figcaption>
        </figure>
      </Transition>
    </div>
    <div class="home-carousel-controls">
      <button type="button" :aria-label="`${label}の前の画像`" @click="goTo(active - 1)">
        <Icon :icon="arrowLeftIcon" aria-hidden="true" />
      </button>
      <span class="home-carousel-count"
        >{{ String(active + 1).padStart(2, '0') }} / {{ String(slides.length).padStart(2, '0') }}</span
      >
      <button type="button" :aria-label="`${label}の次の画像`" @click="goTo(active + 1)">
        <Icon :icon="arrowRightIcon" aria-hidden="true" />
      </button>
      <div class="home-carousel-dots" aria-label="画像を選ぶ">
        <button
          v-for="(_, index) in slides"
          :key="index"
          type="button"
          :aria-label="`${index + 1}枚目の画像を表示`"
          :aria-current="active === index ? 'true' : undefined"
          @click="goTo(index)"
        />
      </div>
    </div>
  </div>
</template>
