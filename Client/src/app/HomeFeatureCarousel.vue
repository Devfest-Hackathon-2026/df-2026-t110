<script setup lang="ts">
import { ref } from 'vue'
import { Carousel, Slide } from 'vue3-carousel'
import 'vue3-carousel/carousel.css'

interface FeatureSlide {
  image: string
  alt: string
  caption: string
}

const props = defineProps<{ label: string; slides: FeatureSlide[] }>()
const active = ref(0)

function goTo(index: number) {
  active.value = (index + props.slides.length) % props.slides.length
}
</script>

<template>
  <div class="home-carousel" role="region" aria-roledescription="スライドショー" :aria-label="label">
    <Carousel
      v-model="active"
      class="home-carousel-view"
      :items-to-show="1.16"
      breakpoint-mode="carousel"
      :breakpoints="{ 700: { itemsToShow: 1.6 } }"
      snap-align="center"
      :wrap-around="true"
      :transition="400"
      :i18n="{ ariaGallery: label, itemXofY: '{slidesCount}枚中{currentSlide}枚目' }"
    >
      <Slide v-for="(slide, index) in slides" :key="slide.image" class="home-carousel-slide">
        <button
          type="button"
          class="home-carousel-image"
          :aria-label="`${index + 1}枚目の画像を表示: ${slide.alt}`"
          :aria-current="active === index ? 'true' : undefined"
          @click="goTo(index)"
        >
          <img :src="slide.image" :alt="slide.alt" width="1200" height="640" />
        </button>
      </Slide>
    </Carousel>
    <p class="home-carousel-caption furigana-text" aria-live="polite">{{ slides[active]?.caption }}</p>
    <div class="home-carousel-controls">
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
