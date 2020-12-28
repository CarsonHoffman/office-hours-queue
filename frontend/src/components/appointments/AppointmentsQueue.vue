<template>
	<div>
		<div class="block" v-if="admin">
			<p class="title block">Admin Controls</p>
			<appointments-admin-selector
				:queue="queue"
				:loaded="loaded"
				:time="time"
			/>
		</div>
		<transition name="fade" mode="out-in">
			<div
				class="hero is-primary"
				v-if="loaded && queue.schedule.numSlots === 0"
				key="unavailable"
			>
				<div class="hero-body">
					<font-awesome-icon icon="frown-open" size="10x" class="block" />
					<h1 class="title block">
						There are no appointments available today.
					</h1>
					<h2 class="subtitle">
						Distance makes the heart grow fonder&hellip;or something like that.
					</h2>
				</div>
			</div>
			<div class="columns" v-else key="other">
				<div class="column is-12">
					<h1 class="title block">Sign Up</h1>
					<appointments-sign-up
						class="block"
						:queue="queue"
						:loaded="loaded"
						:time="time"
					/>
				</div>
			</div>
		</transition>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import ErrorDialog from '@/util/ErrorDialog';
import { AppointmentsQueue } from '@/types/AppointmentsQueue';
import AppointmentsAdminSelector from '@/components/appointments/AppointmentsAdminSelector.vue';
import AppointmentsSignUp from '@/components/appointments/AppointmentsSignUp.vue';

import { library } from '@fortawesome/fontawesome-svg-core';
import { faFrownOpen } from '@fortawesome/free-solid-svg-icons';

library.add(faFrownOpen);

@Component({
	components: {
		AppointmentsSignUp,
		AppointmentsAdminSelector,
	},
})
export default class AppointmentsQueueDisplay extends Vue {
	@Prop({ required: true }) queue!: AppointmentsQueue;
	@Prop({ required: true }) loaded!: boolean;
	@Prop({ required: true }) ws!: WebSocket;
	@Prop({ required: true }) admin!: boolean;
	@Prop({ required: true }) time!: Moment;
}
</script>

<style scoped>
.admin-appointments {
	overflow-x: scroll;
	white-space: nowrap;
}
</style>
