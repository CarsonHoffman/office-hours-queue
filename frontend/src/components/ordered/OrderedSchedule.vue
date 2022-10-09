<template>
	<div class="modal-card" style="width: auto">
		<header class="modal-card-head">
			<p class="modal-card-title">Edit Schedule</p>
			<button type="button" class="delete" @click="$emit('close')" />
		</header>
		<section class="modal-card-body">
			<div class="block" style="padding-bottom: 2em">
				<button class="button timeslot-cell is-danger"></button> Closed
				<button class="button timeslot-cell is-primary"></button> Early Sign Up
				<button class="button timeslot-cell is-success"></button> Open
			</div>
			<div class="block" v-if="schedule !== undefined">
				<div
					v-for="(day, i) in ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']"
					:key="i"
					style="display: block"
				>
					<div class="level">
						<div class="level-left">
							<div class="level-item">
								<p>{{ day }}</p>
							</div>
						</div>
						<div class="level-right">
							<div class="level-item schedule-row">
								<b-tooltip
									:label="
										base
											.clone()
											.add((j - 1) * 30, 'minutes')
											.format('LT')
									"
									v-for="j in 48"
									:key="j"
									:type="classes[i][j - 1]"
									:always="i == 0 && (j - 1) % 8 == 0"
								>
									<button
										class="button timeslot-cell"
										:class="classes[i][j - 1]"
										@mousedown="
											() => {
												changeSlot(i, j - 1);
												painting = true;
											}
										"
										@mouseup="() => (painting = false)"
										@mouseover="() => painting && changeSlot(i, j - 1)"
									></button
								></b-tooltip>
							</div>
						</div>
					</div>
				</div>
			</div>
		</section>
		<footer class="modal-card-foot">
			<button class="button" type="button" @click="$emit('close')">
				Close
			</button>
			<button class="button is-primary" @click="$emit('confirmed', schedule)">
				Save
			</button>
		</footer>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';

@Component({})
export default class OrderedSchedule extends Vue {
	@Prop({ required: true })
	defaultSchedule!: string[];

	schedule = this.defaultSchedule;

	painting = false;

	base: Moment = moment()
		.tz('America/New_York')
		.startOf('day');

	static mappings: { [key: string]: string } = {
		c: 'is-danger',
		p: 'is-primary',
		o: 'is-success',
	};

	static nextState: { [key: string]: string } = {
		c: 'p',
		p: 'o',
		o: 'c',
	};

	get classes() {
		return this.schedule.map((day) =>
			day.split('').map((slot) => OrderedSchedule.mappings[slot])
		);
	}

	changeSlot(i: number, j: number) {
		Vue.set(
			this.schedule,
			i,
			this.schedule[i].slice(0, j) +
				OrderedSchedule.nextState[this.schedule[i][j]] +
				this.schedule[i].slice(j + 1)
		);
	}
}
</script>

<style scoped>
.timeslot-cell {
	display: inline-block;
	padding: 0;
	width: 1em;
	height: 1em;
}

.schedule-row {
	padding-left: 3em;
}
</style>
