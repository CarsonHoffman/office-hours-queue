<template>
	<div class="modal-card" style="width: auto">
		<header class="modal-card-head">
			<p class="modal-card-title">Edit Appointments Schedule</p>
			<button type="button" class="delete" @click="$emit('close')" />
		</header>
		<section class="modal-card-body">
			<div class="buttons has-addons">
				<b-button
					v-for="i in 10"
					:key="i"
					@click="selectedNumAppointments = i - 1"
					:active="selectedNumAppointments == i - 1"
					:style="'background-color: ' + getColorForAvailability(i - 1)"
				>
					{{ i - 1 }}
				</b-button>
			</div>
			<div class="block" v-if="schedules !== undefined">
				<div
					class="block"
					v-for="(day, i) in [
						'Sunday',
						'Monday',
						'Tuesday',
						'Wednesday',
						'Thursday',
						'Friday',
						'Saturday',
					]"
					:key="i"
					style="display: block"
				>
					<p class="title is-4">{{ day }}</p>
					<b-field>
						<b-numberinput
							min="5"
							controls-position="compact"
							v-model="temporaryDurations[i]"
						></b-numberinput>
						<p class="control">
							<button
								class="button is-warning"
								@click="setDuration(i)"
								:disabled="temporaryDurations[i] === durations[i]"
							>
								Change Duration
							</button>
							<button
								class="button is-success"
								@click="$emit('saved', i, durations[i], schedules[i])"
								:disabled="
									schedules[i].toString() === defaultSchedules[i].toString()
								"
							>
								Save Schedule
							</button>
						</p></b-field
					>
					<div class="schedule-row">
						<b-tooltip
							:label="
								base
									.clone()
									.add((j - 1) * durations[i], 'minutes')
									.format('LT')
							"
							v-for="j in Math.floor((60 * 24) / durations[i])"
							:key="j"
							:always="(j - 1) % 8 == 0"
						>
							<button
								class="button timeslot-cell"
								:style="
									'background-color: ' +
										getColorForAvailability(schedules[i][j - 1])
								"
								@mousedown="
									() => {
										changeSlot(i, j - 1);
										painting = true;
									}
								"
								@mouseup="() => (painting = false)"
								@mouseover="() => painting && changeSlot(i, j - 1)"
							>
								{{ schedules[i][j - 1] }}
							</button></b-tooltip
						>
					</div>
				</div>
			</div>
		</section>
		<footer class="modal-card-foot">
			<button class="button" type="button" @click="$emit('close')">
				Close
			</button>
		</footer>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';

import { library } from '@fortawesome/fontawesome-svg-core';
import { faMinus, faPlus } from '@fortawesome/free-solid-svg-icons';

library.add(faMinus, faPlus);

// I don't like this component. :)
@Component({})
export default class AppointmentsSchedule extends Vue {
	@Prop({ required: true })
	defaultSchedules!: number[][];

	@Prop({ required: true })
	defaultDurations!: number[];

	schedules: number[][] = [];
	durations: number[] = [];

	// Where durations are stored before they are confirmed.
	temporaryDurations!: number[];

	selectedNumAppointments = 0;
	painting = false;

	// This choice of date is completely arbitrary; I just needed
	// a base time of midnight off which we can add 30-minute intervals.
	base: Moment = moment('2020-01-01T00:00:00-05:00');

	constructor() {
		super();
		for (let i = 0; i < this.defaultSchedules.length; i++) {
			this.schedules.push([...this.defaultSchedules[i]]);
		}
		this.durations = [...this.defaultDurations];
		this.temporaryDurations = [...this.defaultDurations];
	}

	changeSlot(i: number, j: number) {
		Vue.set(this.schedules[i], j, this.selectedNumAppointments);
	}

	setDuration(i: number) {
		const newSchedule = new Array(
			Math.floor((60 * 24) / this.temporaryDurations[i])
		);
		newSchedule.fill(0);
		this.schedules.splice(i, 1, newSchedule);
		this.durations.splice(i, 1, this.temporaryDurations[i]);
	}

	getColorForAvailability(numAvailable: number, brightness = 1) {
		const hueStart = 120;
		const hueMax = 0;
		const hueRange = hueMax - hueStart;
		const maxAvailable = 9;
		if (numAvailable === 0) {
			return '#777';
		}

		const hue = Math.floor(
			hueStart + ((numAvailable - 1) * hueRange) / (maxAvailable - 1)
		);
		return `hsl(${hue}, 39%, ${Math.floor(54 * brightness)}%)`;
	}
}
</script>

<style scoped>
.timeslot-cell {
	display: inline-block;
	padding: 0;
	width: 2em;
	height: 2em;
}

.schedule-row {
	overflow-x: scroll;
	white-space: nowrap;
	padding-bottom: 1em;
	padding-top: 3em;
	padding-left: 3em;
}
</style>
