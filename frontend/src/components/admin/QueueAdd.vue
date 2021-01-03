<template>
	<div class="modal-card" style="width: auto">
		<header class="modal-card-head">
			<p class="modal-card-title">Add Queue</p>
			<button type="button" class="delete" @click="$emit('close')" />
		</header>
		<section class="modal-card-body">
			<b-field label="Name">
				<b-input v-model="name" />
			</b-field>
			<b-field label="Location">
				<b-input v-model="location" />
			</b-field>
			<b-field label="Type (cannot be changed later!)">
				<b-select v-model="type" required>
					<option>ordered</option>
					<option>appointments</option>
				</b-select>
			</b-field>
		</section>
		<footer class="modal-card-foot">
			<button class="button" type="button" @click="$emit('close')">
				Close
			</button>
			<button class="button is-success" type="button" @click="saveQueue">
				Save
			</button>
		</footer>
	</div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';

@Component({})
export default class QueueAdd extends Vue {
	name = '';
	location = '';
	type = '';

	saveQueue() {
		if (this.type === '') {
			this.$buefy.dialog.alert({
				message: 'Please select a queue type.',
				type: 'is-danger',
			});
			return;
		}
		this.$emit('saved', this.name, this.location, this.type);
	}
}
</script>
