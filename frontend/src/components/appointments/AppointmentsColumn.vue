<template>
	<div :style="'width: ' + admin ? '6' : '1.5' + 'em'">
		<div class="time-area">
			<div
				class="time-container"
				v-if="
					admin ||
						index === 0 ||
						timeslot.time.clone().minutes() === 0 ||
						hovering ||
						selected
				"
			>
				<p
					class="time-text"
					:class="{ 'time-highlight': hovering, 'time-selected': selected }"
				>
					{{ timeslot.time.format('LT') }}
				</p>
			</div>
		</div>
		<appointment-cell
			v-for="(s, i) in appointments"
			:key="i"
			:admin="admin"
			:time="time"
			:appointmentSlot="s"
			@selected="$emit('selected', i)"
			@hover="(h) => (hovering = h)"
		/>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import { AppointmentsTimeslot } from '@/types/AppointmentsQueue';
import { AppointmentSlot } from '@/types/Appointment';
import AppointmentCell from '@/components/appointments/AppointmentCell.vue';

@Component({
	components: {
		AppointmentCell,
	},
})
export default class AppointmentsColumn extends Vue {
	@Prop({ required: true }) appointments!: AppointmentSlot[];
	@Prop({ required: true }) index!: number;
	@Prop({ required: true }) time!: Moment;
	@Prop({ required: true }) timeslot!: AppointmentsTimeslot;
	@Prop({ required: true }) selected!: boolean;
	@Prop({ required: true }) admin!: boolean;

	hovering = false;
}
</script>

<style scoped>
.time-area {
	position: relative;
	height: 5em;
}

.time-container {
	position: absolute;
	bottom: 0;
	left: 0;
}

.time-text {
	position: absolute;
	transform: rotate(315deg);
	transform-origin: bottom left;
	left: 1em;
	bottom: 0;
	z-index: 1;
}

.time-highlight {
	font-weight: bold;
}

.time-selected {
	font-weight: bold;
	background-color: #167df0;
	color: white;
	padding: 0 0.25em;
}

.appointment-slots-group {
	display: inline-block;
	padding-right: 1em;
}
</style>
