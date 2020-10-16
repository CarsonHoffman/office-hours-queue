<template>
	<div class="box" v-if="found">
		<section v-if="queue !== null && queue.announcements.length > 0" class="section">
			<h1 class="title">Announcements</h1>
			<div class="block" v-for="announcement in queue.announcements" :key="announcement.id">
				<announcement-display :announcement="announcement" :queue="queue" :admin="admin" />
			</div>
		</section>
		<section class="section" v-if="queue.type === 'ordered'">
			<ordered-queue-display :queue="queue" :loaded="loaded" :ws="ws" :admin="admin" :time="time" />
		</section>
		<section class="section" v-else-if="queue.type === 'appointments'">
			<appointments-queue-display
				:queue="queue"
				:loaded="loaded"
				:ws="ws"
				:admin="admin"
				:time="time"
			/>
		</section>
	</div>
</template>

<script lang="ts">
import Vue from 'vue';
import { Component, Prop } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import Queue from '@/types/Queue';
import OrderedQueue from '@/types/OrderedQueue';
import Announcement from '@/types/Announcement';
import AnnouncementDisplay from '@/components/AnnouncementDisplay.vue';
import OrderedQueueDisplay from '@/components/ordered/OrderedQueue.vue';
import AppointmentsQueueDisplay from '@/components/appointments/AppointmentsQueue.vue';

import { library } from '@fortawesome/fontawesome-svg-core';
import { faFrownOpen } from '@fortawesome/free-solid-svg-icons';

library.add(faFrownOpen);

@Component({
	components: {
		AnnouncementDisplay,
		OrderedQueueDisplay,
		AppointmentsQueueDisplay,
	},
})
export default class QueuePage extends Vue {
	found = false;
	@Prop() loaded = false;
	ws!: WebSocket;
	@Prop() time!: Moment;
	@Prop() timeUpdater!: number;

	created() {
		if (this.queue === undefined) {
			this.$buefy.toast.open({
				duration: 5000,
				message: `I couldn't find that queue! Bringing you back home…`,
				type: 'is-danger',
			});

			this.$router.push('/');
			return;
		}

		this.found = true;

		this.time = moment();
		// We need to manually refresh the time every so often
		// as Vue isn't reactive to moment changes. I don't
		// like doing this either.
		this.timeUpdater = setInterval(() => {
			this.time = moment();
		}, 5 * 1000);

		document.title =
			this.$root.$data.courses[this.$route.params.cid].shortName +
			' Office Hours';

		// Block on WS open so we are connected to receive events
		// *before* getting latest data
		const url = new URL(
			process.env.BASE_URL + `api/queues/${this.queue.id}/ws`,
			window.location.href
		);
		url.protocol = url.protocol.replace('http', 'ws');
		this.ws = new WebSocket(url.href);

		this.ws.onopen = () => {
			this.queue
				.pullQueueInfo(this.time)
				.then(() => (this.loaded = true))
				.then(() => console.log(this.queue));
			/* this.queue */
			/*   .pullQueueInfo() */
			/*   .then(() => setTimeout(() => (this.loaded = true), 5000)); */
		};

		this.ws.onclose = (c) => {
			if (c.code !== 1005) {
				console.log('disconnected:');
				console.log(c);
				this.$buefy.toast.open({
					duration: 5000,
					message:
						'It looks like you got disconnected from the server. Refreshing…',
					type: 'is-danger',
				});
				this.$emit('disconnected');
			}
		};

		this.ws.onmessage = (e) => {
			const msg = JSON.parse(e.data);
			const type = msg.e;
			const data = msg.d;

			this.queue.handleWSMessage(type, data, this.ws);
		};
	}

	destroyed() {
		if (this.ws !== undefined) {
			this.ws.close();
		}

		clearInterval(this.timeUpdater);
	}

	get queue() {
		return this.$root.$data.queues[this.$route.params.qid];
	}

	get admin() {
		return (
			this.$root.$data.userInfo !== undefined &&
			this.$root.$data.userInfo.admin_courses !== undefined &&
			this.$root.$data.userInfo.admin_courses.includes(this.$route.params.cid)
		);
	}
}
</script>
