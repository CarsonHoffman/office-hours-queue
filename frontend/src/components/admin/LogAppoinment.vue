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
import { defineComponent } from '@vue/composition-api';

export default defineComponent({
	setup() {},
});
</script>

