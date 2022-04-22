<template>
	<div class="modal-card" style="width: auto">
		<header class="modal-card-head">
			<p class="modal-card-title">Log Appointments</p>
			<button type="button" class="delete" @click="$emit('close')" />
		</header>
		<section class="modal-card-body">
			<div class="block">
				<p class="title">Appoinments Log</p>
				<p>
					<b> Date | # of available appoinment | # of used left </b>
				</p>
				<ui
					style="list-style-type: none"
					v-for="(appoinment, i) in appoinments_log"
					:key="i"
				>
					{{ appoinment.date, }} &#9; | &#9; {{ appoinment.available }} &#9; |
					&#9; &#9;
					{{ appoinment.used }}
				</ui>
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

import Appoinmentlog from '../../types/Appointmentlog';

@Component({})
export default class LogAppoinment extends Vue {
	@Prop({ required: true }) readonly appointment_log!: Appoinmentlog;
	log_data = '';

	data() {
		return {
			appoinments_log: this.appointment_log,
		};
	}
}
</script>

