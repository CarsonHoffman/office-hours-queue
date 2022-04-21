<template>
	<div class="modal-card" style="width: auto">
		<header class="modal-card-head">
			<p class="modal-card-title">Log Appointments</p>
			<button type="button" class="delete" @click="$emit('close')" />
		</header>
		<section class="modal-card-body">
			<div class="block">
				<p class="title">Appoinments Log</p>
			</div>
		</section>
		<footer class="modal-card-foot">
			<button class="button" type="button" @click="$emit('close')">
				Close
			</button>
		</footer>
	</div>
</template>


<script>
import { Component, Prop, Vue } from 'vue-property-decorator';

@Component({})
export default class LogAppoinment extends Vue {

	@Prop({ required: true }) log_data!: string[];

	log_data = '';
	date = '';
	appoinments_used = '';
	appoinment_available = '';

	/*
	data() {
		return {
			log_data: {},
		};
	}
	*/
}
</script>

