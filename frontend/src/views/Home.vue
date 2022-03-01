<template>
	<div class="hero is-primary">
		<div class="hero-body">
			<font-awesome-icon
				v-if="mobile"
				icon="hand-point-up"
				size="10x"
				class="block"
			/>
			<font-awesome-icon
				v-else
				icon="hand-point-left"
				size="10x"
				class="block"
			/>
			<h1 class="title block">Welcome to EECS Office Hours!</h1>
			<h2 class="subtitle" v-if="mobile">Select a course above to begin.</h2>
			<h2 class="subtitle" v-else>Select a course on the left to begin.</h2>
		</div>
	</div>
</template>

<script lang="ts">
import Vue from 'vue';
import { Component, Prop } from 'vue-property-decorator';

import { library } from '@fortawesome/fontawesome-svg-core';
import {
	faHandPointUp,
	faHandPointLeft,
} from '@fortawesome/free-solid-svg-icons';

library.add(faHandPointUp, faHandPointLeft);

@Component
export default class HomePage extends Vue {
	windowWidth = window.innerWidth;

	created() {
		this.$root.$data.showCourses = true;
	}

	mounted() {
		window.addEventListener('resize', () => {
			this.windowWidth = window.innerWidth;
		});
	}

	get mobile() {
		return this.windowWidth <= 768;
	}
}
</script>
