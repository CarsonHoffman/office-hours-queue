<template>
	<div class="box" v-if="found">
		<div style="position: relative">
			<div class="buttons top-right" v-if="admin">
				<b-tooltip label="Number of active connections to this queue">
					<button class="button is-white no-hover" v-if="queue !== null">
						<span class="icon"><font-awesome-icon icon="ethernet"/></span>
						<span
							><b>{{ queue.websocketConnections }}</b></span
						>
					</button></b-tooltip
				>
				<button class="button is-light" @click="openManageDialog">
					<span class="icon"><font-awesome-icon icon="cog"/></span>
					<span>Manage Queue</span>
				</button>
			</div>
			<section
				v-if="queue !== null && queue.announcements.length > 0"
				class="section"
			>
				<h1 class="title">Announcements</h1>
				<div
					class="block"
					v-for="announcement in queue.announcements"
					:key="announcement.id"
				>
					<announcement-display
						:announcement="announcement"
						:queue="queue"
						:admin="admin"
					/>
				</div>
			</section>
			<section class="section" v-if="queue.type === 'ordered'">
				<ordered-queue-display
					:queue="queue"
					:loaded="loaded"
					:ws="ws"
					:admin="admin"
					:time="time"
				/>
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
import QueueManage from '@/components/admin/QueueManage.vue';
import ErrorDialog from '@/util/ErrorDialog';

import { library } from '@fortawesome/fontawesome-svg-core';
import { faCog, faEthernet } from '@fortawesome/free-solid-svg-icons';

library.add(faCog, faEthernet);

@Component({
	components: {
		AnnouncementDisplay,
		OrderedQueueDisplay,
		AppointmentsQueueDisplay,
	},
})
export default class QueuePage extends Vue {
	@Prop({ required: true }) studentView!: boolean;

	found = false;
	loaded = false;
	ws!: WebSocket;
	lastMessage: Moment | undefined;
	time = moment();
	timeUpdater!: number;

	created() {
		this.$root.$data.showCourses = false;

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

		// We need to manually refresh the time every so often
		// as Vue isn't reactive to moment changes. I don't
		// like doing this either.
		this.timeUpdater = window.setInterval(() => {
			this.time = moment();
			if (
				this.lastMessage !== undefined &&
				this.time.diff(this.lastMessage, 'seconds') > 10 + 2
			) {
				location.reload();
			}
		}, 5 * 1000);

		document.title = this.queue.course.shortName + ' Office Hours';

		// Block on WS open so we are connected to receive events
		// *before* getting latest data
		const url = new URL(
			process.env.BASE_URL + `api/queues/${this.queue.id}/ws`,
			window.location.href
		);
		url.protocol = url.protocol.replace('http', 'ws');
		this.ws = new WebSocket(url.href);

		this.ws.onopen = () => {
			this.queue.pullQueueInfo(this.time).then(() => (this.loaded = true));
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
			this.lastMessage = moment();

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
		return this.$root.$data.queues[this.$route.params.qid] || null;
	}

	get admin() {
		return (
			!this.studentView &&
			this.$root.$data.userInfo !== undefined &&
			this.$root.$data.userInfo.admin_courses !== undefined &&
			this.$root.$data.userInfo.admin_courses.includes(this.queue.course.id)
		);
	}

	openManageDialog() {
		Promise.all([
			fetch(process.env.BASE_URL + `api/queues/${this.queue.id}/configuration`),
			fetch(process.env.BASE_URL + `api/queues/${this.queue.id}/groups`),
		])
			.then(([config, groups]) => Promise.all([config.json(), groups.json()]))
			.then(([configuration, groups]) => {
				this.$buefy.modal.open({
					parent: this,
					component: QueueManage,
					props: {
						defaultConfiguration: configuration,
						defaultGroups: groups,
						type: this.queue.type,
					},
					events: {
						configurationSaved: (newConfiguration: {
							[index: string]: any;
						}) => {
							fetch(
								process.env.BASE_URL +
									`api/queues/${this.queue.id}/configuration`,
								{
									method: 'PUT',
									body: JSON.stringify(newConfiguration),
								}
							).then((res) => {
								if (res.status !== 204) {
									return ErrorDialog(res);
								}
								this.$buefy.toast.open({
									duration: 5000,
									message: 'Queue settings saved!',
									type: 'is-success',
								});
							});
						},
						groupsSaved: (newGroups: string[][]) => {
							fetch(
								process.env.BASE_URL + `api/queues/${this.queue.id}/groups`,
								{
									method: 'PUT',
									body: JSON.stringify(newGroups),
								}
							).then((res) => {
								if (res.status !== 204) {
									return ErrorDialog(res);
								}
								this.$buefy.toast.open({
									duration: 5000,
									message: 'Queue groups saved!',
									type: 'is-success',
								});
							});
						},
						announcementAdded: (content: string) => {
							fetch(
								process.env.BASE_URL +
									`api/queues/${this.queue.id}/announcements`,
								{ method: 'POST', body: JSON.stringify({ content: content }) }
							).then((res) => {
								if (res.status !== 201) {
									return ErrorDialog(res);
								}
								this.$buefy.toast.open({
									duration: 5000,
									message: 'Announcement added!',
									type: 'is-success',
								});
							});
						},
					},
					hasModalCard: true,
					trapFocus: true,
				});
			});
	}
}
</script>

<style scoped>
.top-right {
	position: absolute;
	top: 0;
	right: 0;
}

.no-hover {
	pointer-events: none;
}

.section {
	padding: 3rem 1.5rem;
}
</style>
