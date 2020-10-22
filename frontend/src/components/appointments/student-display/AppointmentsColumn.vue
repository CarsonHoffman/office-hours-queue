<template>
	<div>
		<div class="time-area">
			<div
				class="time-container"
				v-if="index === 0 || slot.time.clone().minutes() === 0 || hovering || selected"
			>
				<p
					class="time-text"
					:class="{'time-highlight': hovering, 'time-selected': selected}"
				>{{slot.time.format('LT')}}</p>
			</div>
		</div>
		<button
			class="button appointment-cell"
			v-for="i in slot.total"
			:key="i"
			:class="getClasses(i-1)"
			:disabled="(past || taken(i-1)) && !(myAppointment && i === 1)"
			@mouseover="hovering = true"
			@mouseleave="hovering = false"
			@click="$emit('selected')"
		></button>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import { AppointmentsTimeslot } from '@/types/AppointmentsQueue';

@Component
export default class AppointmentsColumn extends Vue {
	@Prop({ required: true }) slot!: AppointmentsTimeslot;
	@Prop({ required: true }) index!: number;
	@Prop({ required: true }) time!: Moment;
	@Prop({ required: true }) selected!: boolean;

	@Prop({ required: true }) myAppointment!: boolean;

	hovering = false;

	get past() {
		return this.slot.past(this.time);
	}

	taken(index: number) {
		return index < this.slot.studentSlots.length;
	}

	getClasses(index: number) {
		return {
			'is-success': !this.taken(index),
			'is-danger': this.taken(index) && !(this.myAppointment && index === 0),
			'is-primary': this.myAppointment && index === 0,
			'is-light': this.past,
		};
	}
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

.appointment-cell {
	display: block;
	margin-bottom: 2px;
	padding: 0;
	width: 1.5em;
	height: 1.5em;
}
</style>
