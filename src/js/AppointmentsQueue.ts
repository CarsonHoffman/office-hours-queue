import { Page } from './Queue';
import { Course, QueueApplication, User } from './QueueApplication';
import { oops, showErrorMessage, Mutable, assert } from './util/util';
import $ from 'jquery';
import moment, { duration, Moment } from 'moment-timezone';
import {
    MessageResponses,
    messageResponse,
    Observable,
    addListener,
    Message,
} from './util/mixins';
import {
    SignUpForm,
    SignUpMessage,
    AppointmentSchedule,
    Appointment,
    filterAppointmentsSchedule,
} from './OrderedQueue';
import 'jquery.scrollto';

export function extractScheduleFromResponse(scheduleData: any) {
    let duration: number = scheduleData['duration'];
    let scheduledTime = moment().tz('America/New_York').startOf('day');
    let now = moment();
    let schedule: AppointmentSchedule = (<string>scheduleData['schedule'])
        .split('')
        .map((n, i) => {
            let slots = {
                timeslot: i,
                duration: duration,
                scheduledTime: scheduledTime.clone(),
                numAvailable: parseInt(n),
                numFilled: 0,
            };
            scheduledTime = scheduledTime.add(duration, 'm');
            return slots;
        });
    return schedule;
}

export class AppointmentsQueue {
    public readonly kind = 'appointments';

    public readonly observable = new Observable(this);
    public readonly page: Page;

    public readonly myRequest: Appointment | null = null;

    private readonly elem: JQuery;
    private readonly adminControlsElem: JQuery;
    private readonly studentControlsElem: JQuery;
    private readonly appointmentsElem: JQuery;
    // private readonly stackElem: JQuery;

    private readonly adminControls: AdminControls;
    private readonly studentControls: StudentControls;
    public readonly schedule?: AppointmentSchedule;

    constructor(data: { [index: string]: any }, page: Page, elem: JQuery) {
        this.page = page;
        this.elem = elem;

        this.adminControlsElem = $(
            '<div class="panel panel-default adminOnly"><div class="panel-body"></div></div>',
        )
            .appendTo(this.elem)
            .find('.panel-body');

        this.adminControls = new AdminControls(this, this.adminControlsElem);

        this.studentControlsElem = $(
            '<div class="panel panel-default"><div class="panel-body"></div></div>',
        )
            .appendTo(this.elem)
            .find('.panel-body');

        this.studentControls = new StudentControls(
            this,
            this.studentControlsElem,
        );
        this.observable.addListener(this.studentControls);

        this.appointmentsElem = $('<div></div>').appendTo(elem);
    }

    // $.getJSON(`api/queues/${this.queueId}/appointments/0`).then((a) => console.log(JSON.stringify(a, null, 4)));
    // $.getJSON(`api/queues/${this.queueId}/appointmentsSchedule`).then((a) => console.log(JSON.stringify(a, null, 4)));

    public refreshRequest() {
        let email = User.email();
        // if (!email) {
        //     return;
        // }
        let day = moment().tz('America/New_York').day();
        let calls = [
            $.ajax({
                type: 'GET',
                url: `api/queues/${this.page.queueId}/appointments/${day}`,
                dataType: 'json',
            }),
            $.ajax({
                type: 'GET',
                url: `api/queues/${this.page.queueId}/appointments/schedule/${day}`,
                dataType: 'json',
            }),
        ];

        if (email) {
            calls.push(
                $.ajax({
                    type: 'GET',
                    url: `api/queues/${this.page.queueId}/appointments/${day}/@me`,
                    dataType: 'json',
                }),
            );
        }

        return Promise.all(calls);
    }

    public refreshResponse(data: any) {
        let scheduleData = data[1];
        let duration: number = scheduleData['duration'];
        let padding: number = scheduleData['padding'];
        let scheduledTime = moment().tz('America/New_York').startOf('day');
        let totalAvailable = 0;
        let schedule = extractScheduleFromResponse(scheduleData);
        let totalStillAvailable = schedule.reduce(
            (prev, current) =>
                current.scheduledTime.diff(now) > 0
                    ? prev + current.numAvailable
                    : prev,
            0,
        );
        (<Mutable<this>>this).schedule = schedule;

        // let appointments : any[][] = [];
        // for(let i = 0; i < schedule.length; ++i) {
        //     appointments.push([]);
        // }
        let appointmentsData: any[] = data[0];
        let startOfDay = moment().tz('America/New_York').startOf('day');
        let now = moment();
        let totalFilled = 0;
        let totalStillFilled = 0;
        let appointments: Appointment[] = appointmentsData.map(
            (appData: any) => {
                let appt = createAppointment(appData, startOfDay);
                if (
                    appt.staffEmail === undefined ||
                    appt.studentEmail !== undefined
                ) {
                    ++schedule[appt.timeslot].numFilled;
                    ++totalFilled;
                    if (appt.scheduledTime.diff(now) > 0) {
                        ++totalStillFilled;
                    }
                }

                return appt;
            },
        );

        this.page.setStatusMessage(
            `${
                totalStillAvailable - totalStillFilled
            }/${totalStillAvailable} appointments available below!`,
        );

        if (data.length > 2) {
            let myAppointments: Appointment[] = data[2];
            let myAppointment: Appointment | null = null;
            myAppointments
                .map((appData) => createAppointment(appData, startOfDay))
                .forEach((appt) => {
                    if (
                        !myAppointment &&
                        appt.studentEmail !== undefined &&
                        User.isMe(appt.studentEmail) &&
                        now.diff(appt.scheduledTime, 'minutes') < appt.duration
                    ) {
                        myAppointment = appt;
                    }
                });
            this.setMyAppointment(myAppointment);
        }

        this.adminControls.setAppointments(schedule, appointments);
        this.studentControls.setAppointments(schedule);

        this.observable.send('queueRefreshed');

        // let queue = data["queue"];

        // this.page.setNumEntries(this.numEntries);
    }

    public setMyAppointment(myRequest: Appointment | null) {
        (<Appointment | null>this.myRequest) = myRequest;
        this.studentControls.setMyAppointment();
    }

    // public clear() {
    //     // Do nothing
    // }

    public signUp(
        name: string,
        location: string,
        description: string,
        mapX: number,
        mapY: number,
        timeslot: number,
    ) {
        if (!this.schedule || this.schedule.length === 0) {
            return;
        }

        let scheduledTime = moment().tz('America/New_York').startOf('day');
        scheduledTime.add(timeslot * this.schedule[0].duration, 'm');

        return $.ajax({
            type: 'POST',
            url: `api/queues/${
                this.page.queueId
            }/appointments/${scheduledTime.day()}/${timeslot}`,
            data: JSON.stringify({
                name: name,
                location: location,
                map_x: mapX,
                map_y: mapY,
                description: description,
            }),
            dataType: 'json',
            success: (data) => {
                this.page.refresh();
            },
            error: oops,
        });
    }

    public updateRequest(
        name: string,
        location: string,
        description: string,
        mapX: number,
        mapY: number,
        timeslot: number,
    ) {
        if (!this.schedule || this.schedule.length === 0) {
            return;
        }

        if (!this.myRequest) {
            return;
        }

        if (
            this.myRequest.scheduledTime.diff(moment()) < 0 &&
            timeslot !== this.myRequest.timeslot
        ) {
            showErrorMessage(
                "You can't move an appointment that has already happened!",
            );
            return;
        }

        return $.ajax({
            type: 'PUT',
            url: `api/queues/${this.page.queueId}/appointments/${this.myRequest.id}`,
            data: JSON.stringify({
                name: name,
                timeslot: timeslot,
                location: location,
                mapX: mapX,
                mapY: mapY,
                description: description,
            }),
            contentType: 'application/json',
            success: (data) => {
                this.page.refresh();
            },
            error: oops,
        });
    }

    public removeAppointment(app: Appointment) {
        console.log(
            'attempting to remove ' +
                app.studentEmail +
                ' from queue ' +
                this.page.queueId,
        );
        this.page.disableRefresh();

        if (app.scheduledTime.diff(moment()) <= 0) {
            showErrorMessage(
                "You can't delete an appointment that has already happened!",
            );
            this.page.enableRefresh();
            return;
        }

        $.ajax({
            type: 'DELETE',
            url: `api/queues/${this.page.queueId}/appointments/${app.id}`,
            success: () => {
                console.log(
                    'successfully removed ' +
                        app.studentEmail +
                        ' from queue ' +
                        this.page.queueId,
                );
            },
            error: oops,
        }).always(() => {
            setTimeout(() => {
                this.page.enableRefresh();
                this.page.refresh();
            }, 100);
        });
    }
}

class AppointmentViewer {
    public readonly observable = new Observable(this);

    public readonly queue: AppointmentsQueue;
    public readonly selected: Appointment | null = null;
    public readonly claimed: readonly Appointment[] = [];

    private elem: JQuery;
    private appointmentDetailsElem: JQuery;
    private emailElem: JQuery;
    private nameElem: JQuery;
    private descriptionElem: JQuery;
    private locationElem: JQuery;
    private claimButton: JQuery;
    private releaseButton: JQuery;

    public constructor(queue: AppointmentsQueue, elem: JQuery) {
        this.queue = queue;
        this.elem = elem;

        this.appointmentDetailsElem = $('<div></div>')
            .appendTo(this.elem)
            .hide()
            .append('<span>email: </span>')
            .append((this.emailElem = $('<span></span>')))
            .append('<br />')
            .append('<span>name: </span>')
            .append((this.nameElem = $('<span></span>')))
            .append('<br />')
            .append('<span>description: </span>')
            .append((this.descriptionElem = $('<span></span>')))
            .append('<br />')
            .append('<span>location: </span>')
            .append((this.locationElem = $('<span></span>')));

        let buttonsElem = $('<div></div>')
            .appendTo(this.appointmentDetailsElem)
            .append(
                (this.claimButton = $(
                    $(
                        '<button type="button" class="btn btn-success adminOnly">Claim</button>',
                    ),
                )),
            )
            .append(' ')
            .append(
                (this.releaseButton = $(
                    $(
                        '<button type="button" class="btn btn-warning adminOnly">Release</button>',
                    ),
                )),
            );

        this.claimButton.click(() => this.claimSelectedAppointment());
        this.releaseButton.click(
            () =>
                this.selected && this.releaseClaimedAppointment(this.selected),
        );
    }

    private claimSelectedAppointment() {
        if (!this.selected) {
            return;
        }

        const day = moment().tz('America/New_York').day();
        $.ajax({
            type: 'PUT',
            url: `api/queues/${this.queue.page.queueId}/appointments/${day}/claims/${this.selected.timeslot}`,
            success: (data) => {
                this.queue.page.refresh();
            },
            error: oops,
        }).always(() => {
            setTimeout(() => {
                this.queue.page.enableRefresh();
                this.queue.page.refresh();
            }, 100);
        });
    }

    private releaseClaimedAppointment(appt: Appointment) {
        const day = moment().tz('America/New_York').day();
        $.ajax({
            type: 'DELETE',
            url: `api/queues/${this.queue.page.queueId}/appointments/claims/${appt.id}`,
            success: (data) => {
                this.queue.page.refresh();
            },
            error: oops,
        }).always(() => {
            setTimeout(() => {
                this.queue.page.enableRefresh();
                this.queue.page.refresh();
            }, 100);
        });
    }

    public setSelectedAppointment(appt: Appointment | null) {
        (<Mutable<this>>this).selected = appt;
        if (appt) {
            this.appointmentDetailsElem.show();
            this.emailElem.html(
                appt.studentEmail || `(no student, timeslot ${appt.timeslot})`,
            );
            this.nameElem.html(appt.name || '(no student)');
            this.descriptionElem.html(appt.description || '(no student)');
            this.locationElem.html(appt.location || '(no student)');

            if (!appt.staffEmail) {
                // unclaimed
                this.claimButton.show().html('Claim').prop('disabled', false);
                this.releaseButton.hide();
            } else if (appt.staffEmail === User.email()) {
                // claimed by us
                this.claimButton
                    .show()
                    .html(
                        "<span class='glyphicon glyphicon-ok'></span> Claimed by You",
                    )
                    .prop('disabled', true);
                this.releaseButton.show();
            } else {
                // claimed by someone else
                this.claimButton
                    .show()
                    .html(`Claimed by ${appt.staffEmail}`)
                    .prop('disabled', true);
                this.releaseButton.hide();
            }
        } else {
            this.appointmentDetailsElem.hide();
        }
    }

    public setClaimedAppointments(claimed: Appointment[]) {
        (<Mutable<this>>this).claimed = claimed;
    }
}

const dayLetters = ['S', 'M', 'T', 'W', 'T', 'F', 'S'];
const dayNames = [
    'Sunday',
    'Monday',
    'Tuesday',
    'Wednesday',
    'Thursday',
    'Friday',
    'Saturday',
];
const hueStart = 120;
const hueMax = 0;
const hueRange = hueMax - hueStart;
const maxAvailable = 9;

function getColorForAvailability(numAvailable: number, brightness: number = 1) {
    if (numAvailable === 0) {
        return '#777';
    }

    let hue = Math.floor(
        hueStart + ((numAvailable - 1) * hueRange) / (maxAvailable - 1),
    );
    return `hsl(${hue}, 39%, ${Math.floor(54 * brightness)}%)`;
}

export class AppointmentsSchedulePicker {
    // private readonly unitElems : JQuery[][];

    private readonly schedules: AppointmentSchedule[] = [
        [],
        [],
        [],
        [],
        [],
        [],
        [],
    ];

    private dialog: JQuery;
    private readonly pickerTables: readonly JQuery[];
    private readonly slotsElems: JQuery[][] = [[], [], [], [], [], [], []];
    private currentAvailabilityBrush: number = 0;

    constructor() {
        let dialog = (this.dialog = $('#appointmentsScheduleDialog'));

        let pickerContainer = $('#appt-schedule-picker-container');
        this.pickerTables = dayNames.map((dayName, i) => {
            $(`<h3>${dayName}</h3>`).appendTo(pickerContainer);

            let form = $(`<form class="appointmentsDurationForm form-inline" role="form">
            <input type="hidden" name="day" value="${i}" />
            <div class="input-group">
                <span class="input-group-addon" id="appointmentsDurationInputLabel${i}">Duration</span>
                
                <input id="appointmentsDurationInput${i}" type="number" required name="duration" class="form-control" min="5" max="120" maxlength="3" aria-describedby="appointmentsDurationInputLabel">
                <span class="input-group-btn">
                    <button class="btn btn-primary" type="submit">Adjust</button>
                </span>
            </div>
        </form>`).appendTo(pickerContainer);

            $(
                `<button type="button" disabled class="updateAppointmentSlotsButton btn btn-success">Update Appointment Slots</button>`,
            )
                .appendTo(form)
                .click(() => {
                    this.updateSchedule(i);
                });

            return $('<table></table>').appendTo(
                $(`<div class="appt-schedule-picker"></div>`).appendTo(
                    pickerContainer,
                ),
            );
        });

        let self = this;
        $('.appointmentsDurationForm').submit(function (e) {
            e.preventDefault();

            let day = $(this).find('input[name=day]').val();
            let duration = $(this).find('input[name=duration]').val();
            if (typeof day === 'string' && typeof duration === 'string') {
                self.changeDuration(parseInt(day), parseInt(duration));
            }

            return false;
        });

        $('#appointmentsScheduleForm').submit((e) => {
            e.preventDefault();

            // this.update();

            dialog.modal('hide');
            return false;
        });

        dialog.on('shown.bs.modal', () => {
            this.refresh();
        });

        for (let i = 0; i < 10; ++i) {
            $('#appointmentsScheduleNumberButtons').append(
                $(
                    `<button type="button" class="btn" style="color: white; background-color: ${getColorForAvailability(
                        i,
                    )}; border-color: ${getColorForAvailability(
                        i,
                        0.8,
                    )};">${i}</button>`,
                ).click(() => {
                    this.currentAvailabilityBrush = i;
                    $('#appointmentsScheduleNumberButtons')
                        .find('button')
                        .removeClass('active')
                        .eq(i)
                        .addClass('active');
                }),
            );
        }
        $('#appointmentsScheduleNumberButtons')
            .find('button')
            .first()
            .addClass('active');
    }

    private changeDuration(day: number, duration: number) {
        let originalNumTimeslots = this.schedules[day].length;
        let newNumTimeslots = Math.floor((24 * 60) / duration);
        let timeslotsRatio = originalNumTimeslots / newNumTimeslots;
        let originalSchedule = this.schedules[day];
        let newSchedule: AppointmentSchedule = [];
        let scheduledTime = moment().tz('America/New_York').startOf('day');
        for (let t = 0; t < newNumTimeslots; ++t) {
            newSchedule.push({
                duration: duration,
                numAvailable:
                    originalSchedule[Math.floor(t * timeslotsRatio)]
                        .numAvailable,
                numFilled: 0, // not used
                scheduledTime: scheduledTime.clone(),
                timeslot: t,
            });
            scheduledTime = scheduledTime.add(duration, 'm');
        }
        this.setSchedule(day, newSchedule);
        this.enableUpdateButton(day);
    }

    private enableUpdateButton(day: number) {
        $('.updateAppointmentSlotsButton')
            .eq(day)
            .html('Update Appointment Slots')
            .addClass('btn-warning')
            .removeClass('btn-success')
            .prop('disabled', false);
    }

    public setSchedule(day: number, schedule: AppointmentSchedule) {
        assert(schedule.length !== 0); // This should not be a filtered schedule, so its length must be nonzero
        this.schedules[day] = schedule;

        $(`#appointmentsDurationInput${day}`).val(schedule[0].duration);

        let pickerTable = this.pickerTables[day];
        pickerTable.empty();

        // First row of table with time headers
        let firstRow = $('<tr></tr>').appendTo(pickerTable);
        let secondRow = $('<tr></tr>').appendTo(pickerTable);

        // firstRow.append(`<td rowspan="2" style="margin-right: 3px;">${dayNames[day]}</td>`);

        this.slotsElems[day] = schedule.map((slots) => {
            firstRow.append(
                `<th class="appointment-slots-header"><span>${slots.scheduledTime.format(
                    'h:mma',
                )}</span></th>`,
            );
            let label = $(
                `<span class="label" style="background-color: ${getColorForAvailability(
                    slots.numAvailable,
                )};">${slots.numAvailable}</span>`,
            );
            label.data('timeslot', slots.timeslot);
            // label.data("numAvailable", slots.numAvailable);
            secondRow.append(
                $(`<td></td>`).append(
                    $('<div class="appt-schedule-picker-slot"></div>').append(
                        label,
                    ),
                ),
            );
            return label;
        });

        // this.unitElems = [];
        // for(var r = 0; r < 7; ++r) {
        //     var day : JQuery[] = [];
        //     var rowElem = $('<tr></tr>');
        //     rowElem.append('<td style="width:1em; text-align: right; padding-right: 3px;">' + dayLetters[r] + '</td>');
        //     for(var c = 0; c < 48; ++c) {
        //         var unitElem = $('<td><div class="scheduleUnit"></div></td>').appendTo(rowElem).find(".scheduleUnit");
        //         day.push(unitElem);
        //     }
        //     this.unitElems.push(day);
        //     this.slotsTable.append(rowElem);
        // }

        let pressed = false;
        pickerTable.on('mousedown', function (e) {
            e.preventDefault();
            pressed = true;
            return false;
        });
        pickerTable.on('mouseup', function () {
            pressed = false;
        });
        pickerTable.on('mouseleave', function () {
            pressed = false;
        });
        this.dialog.on('hidden.bs.modal', function () {
            pressed = false;
        });

        let changeNumAvailable = (elem: JQuery) => {
            if (pressed) {
                // elem.data("numAvailable", this.currentAvailabilityBrush);
                elem.css(
                    'background-color',
                    getColorForAvailability(this.currentAvailabilityBrush),
                );
                elem.html('' + this.currentAvailabilityBrush);

                if (
                    this.schedules[day][elem.data('timeslot')].numAvailable !==
                    this.currentAvailabilityBrush
                ) {
                    this.schedules[day][
                        elem.data('timeslot')
                    ].numAvailable = this.currentAvailabilityBrush;
                    this.enableUpdateButton(day);
                }
            }
        };
        pickerTable.on(
            'mouseover',
            '.appt-schedule-picker-slot > .label',
            function (e) {
                e.preventDefault();
                changeNumAvailable($(this));
                return false;
            },
        );
        pickerTable.on(
            'mousedown',
            '.appt-schedule-picker-slot > .label',
            function (e) {
                e.preventDefault();
                pressed = true;
                changeNumAvailable($(this));
                return false;
            },
        );
    }

    public refresh() {
        let aq = QueueApplication.instance.activeQueue()?.queue;

        if (aq?.kind === 'appointments') {
            return $.ajax({
                type: 'GET',
                url: `api/queues/${aq.page.queueId}/appointments/schedule`,
                dataType: 'json',
                success: (data: any[]) => {
                    // data is an array of the schedules
                    data.forEach((schedule, i) =>
                        this.setSchedule(
                            i,
                            extractScheduleFromResponse(schedule),
                        ),
                    );
                    $('.updateAppointmentSlotsButton')
                        .html(
                            '<span class="glyphicon glyphicon-ok"></span> Up To Date',
                        )
                        .addClass('btn-success')
                        .removeClass('btn-warning')
                        .prop('disabled', true);
                },
                error: oops,
            });
        }
    }

    public updateSchedule(day: number) {
        let aq = QueueApplication.instance.activeQueue()?.queue;

        if (aq?.kind === 'appointments') {
            let schedule = this.schedules[day]
                .map((slots) => slots.numAvailable)
                .join('');

            return $.ajax({
                type: 'PUT',
                url: `api/queues/${aq.page.queueId}/appointments/schedule/${day}`,
                data: JSON.stringify({
                    duration: this.schedules[day][0].duration,
                    padding: 2,
                    schedule: schedule,
                }),
                contentType: 'application/json',
                success: (data) => {
                    console.log(`day ${day} schedule updated to ${schedule}`);
                    $('.updateAppointmentSlotsButton')
                        .eq(day)
                        .html(
                            '<span class="glyphicon glyphicon-ok"></span> Up To Date',
                        )
                        .addClass('btn-success')
                        .removeClass('btn-warning')
                        .prop('disabled', true);
                },
                error: oops,
            });
        }
    }
    //     // lol can't make up my mind whether I like functional vs. iterative
    //     var schedule = [];
    //     for(var r = 0; r < 7; ++r) {
    //         schedule.push(this.unitElems[r].map(function(unitElem){
    //             return unitElem.data("scheduleType");
    //         }).join(""));
    //     }

    //     let aq = QueueApplication.instance.activeQueue();
    //     if (aq) {
    //         return $.ajax({
    //             type: "POST",
    //             url: "api/updateSchedule",
    //             data: {
    //                 idtoken: User.idToken(),
    //                 queueId: aq.queueId,
    //                 schedule: schedule
    //             },
    //             success: function() {
    //                 console.log("schedule updated");
    //             },
    //             error: oops
    //         });
    //     }
    // }
}

class AdminControls {
    private queue: AppointmentsQueue;
    private elem: JQuery;
    private appointmentsElem: JQuery;
    private appointmentsTableElem: JQuery;
    private headerElems?: JQuery[];
    private headerElemsByTimeslot: { [index: number]: JQuery } = {};
    private crabsterNow: JQuery;
    private crabsterIndex = 0;
    private schedule?: AppointmentSchedule;
    private filteredSchedule?: AppointmentSchedule;
    private appointments?: Appointment[];

    private appointmentViewer: AppointmentViewer;

    private notificationsGiven: { [index: string]: true | undefined } = {};

    public readonly _act!: MessageResponses;

    constructor(queue: AppointmentsQueue, elem: JQuery) {
        this.queue = queue;
        this.elem = elem;

        this.elem.append('<p><b>Admin Controls</b></p>');

        // var clearQueueButton = $('<button type="button" class="btn btn-danger adminOnly" data-toggle="modal" data-target="#clearTheQueueDialog">Clear the queue</button>');
        // this.queue.page.makeActiveOnClick(clearQueueButton); // TODO I don't think this is necessary anymore. If they can click it, it should be active.
        // this.elem.append(clearQueueButton);

        this.elem.append(' ');
        var openScheduleDialogButton = $(
            '<button type="button" class="btn btn-info adminOnly" data-toggle="modal" data-target="#appointmentsScheduleDialog">Edit Appointment Slots</button>',
        );
        this.queue.page.makeActiveOnClick(openScheduleDialogButton); // TODO I don't think this is necessary anymore. If they can click it, it should be active.
        this.elem.append(openScheduleDialogButton);

        // this.elem.append(" ");
        // var openManageQueueDialogButton = $('<button type="button" class="btn btn-info adminOnly" data-toggle="modal" data-target="#manageQueueDialog">Manage Queue</button>');
        // this.queue.page.makeActiveOnClick(openManageQueueDialogButton); // TODO I don't think this is necessary anymore. If they can click it, it should be active.
        // this.elem.append(openManageQueueDialogButton);

        this.elem.append(' ');
        let openAddAnnouncementDialogButton = $(
            '<button type="button" class="btn btn-info adminOnly" data-toggle="modal" data-target="#addAnnouncementDialog">Add Announcement</button>',
        );
        this.queue.page.makeActiveOnClick(openAddAnnouncementDialogButton); // TODO I don't think this is necessary anymore. If they can click it, it should be active.
        this.elem.append(openAddAnnouncementDialogButton);

        this.crabsterNow = $(
            '<div class="crabster-now"><img src="img/crabster_sign.png"></img><span class="crabster-time"></span></div>',
        );
        this.appointmentsElem = $(
            '<div style="overflow-x: scroll"></div>',
        ).append(
            (this.appointmentsTableElem = $(
                '<table style="position: relative"></table>',
            ).append(this.crabsterNow)),
        );

        this.elem.append(' ');
        $(
            '<button type="button" class="btn btn-primary adminOnly">Now</button>',
        )
            .click(() => this.scrollToNow(600))
            .appendTo(this.elem);

        this.elem.append('<p><b>Legend</b></p>');

        this.elem.append(
            $(
                '<button type="button" class="btn btn-default adminOnly">No staff, no student</button>',
            ),
        );
        this.elem.append(' ');
        this.elem.append(
            $(
                '<button type="button" class="btn btn-success adminOnly">Claimed by you, no student</button>',
            ),
        );
        this.elem.append(' ');
        this.elem.append(
            $(
                '<button type="button" class="btn btn-primary adminOnly">Claimed by you, filled by student</button>',
            ),
        );
        this.elem.append(' ');
        this.elem.append(
            $(
                '<button type="button" class="btn btn-warning adminOnly">Claimed by other staff member</button>',
            ),
        );
        this.elem.append(' ');
        this.elem.append(
            $(
                '<button type="button" class="btn btn-danger adminOnly">Filled by student, no staff member</button>',
            ),
        );

        // scroll to now, now and set an interval to scroll to now every 2 minutes
        // this.scrollToNow(0);
        // setInterval(() => this.scrollToNow(600), 120000);

        setInterval(() => this.crawlToNow(3000), 5000);

        this.elem.append(this.appointmentsElem);

        this.appointmentViewer = new AppointmentViewer(
            this.queue,
            $('<div></div>').appendTo(this.elem),
        );
        addListener(this.appointmentViewer, this);
    }

    private scrollToNow(duration: number) {
        if (!this.filteredSchedule || this.filteredSchedule.length === 0) {
            return;
        }
        let schedule = this.filteredSchedule;
        let now = moment();
        let closestIndex = this.filteredSchedule.reduce((prev, current, i) => {
            return Math.abs(current.scheduledTime.diff(now)) <
                Math.abs(schedule[prev].scheduledTime.diff(now))
                ? i
                : prev;
        }, 0);

        if (this.headerElems) {
            this.appointmentsElem.scrollTo(
                this.headerElems[closestIndex],
                duration,
                { easing: 'swing' },
            );
        }
    }

    private crawlToNow(duration: number) {
        if (!this.filteredSchedule || this.filteredSchedule.length === 0) {
            return;
        }
        let schedule = this.filteredSchedule;
        let now = moment();

        let nextIndex = this.crabsterIndex;
        if (nextIndex + 1 >= this.filteredSchedule.length) {
            // NOTE: may be > if schedule got updated and there are now less slots
            nextIndex = 0;
        }
        while (
            nextIndex + 1 < this.filteredSchedule.length &&
            now.diff(this.filteredSchedule[nextIndex + 1].scheduledTime) > 0
        ) {
            ++nextIndex;
        }
        this.crabsterIndex = nextIndex;
        let slot = this.filteredSchedule[this.crabsterIndex];
        // we have passed the start point for that next appointment timeslot
        if (now.isBefore(slot.scheduledTime)) {
            this.crabsterNow
                .find('.crabster-time')
                .html(
                    '<span class="glyphicon glyphicon-arrow-left"></span>' +
                        now.tz('America/New_York').format('h:mm'),
                );
        } else if (
            now.isAfter(
                slot.scheduledTime.clone().add(slot.duration, 'minutes'),
            )
        ) {
            this.crabsterNow
                .find('.crabster-time')
                .html(
                    now.tz('America/New_York').format('h:mm') +
                        '<span class="glyphicon glyphicon-arrow-right"></span>',
                );
        } else {
            this.crabsterNow
                .find('.crabster-time')
                .html(now.tz('America/New_York').format('h:mm'));
        }
        if (this.headerElems) {
            this.crabsterNow.animate(
                {
                    left: this.headerElems[nextIndex].position().left + 'px',
                },
                duration,
            );
        }
    }

    public setAppointments(
        schedule: AppointmentSchedule,
        appointments: Appointment[],
    ) {
        if (!this.queue.page.isAdmin) {
            return;
        }

        let firstTime = false;
        if (!this.appointments) {
            firstTime = true;
        }

        this.schedule = schedule;
        this.appointments = appointments;

        this.appointmentsTableElem.children('tr').remove();

        let maxAppts = schedule.reduce(
            (prev, current) => Math.max(prev, current.numAvailable),
            0,
        );

        // Note: this needs to be done before the filtering below
        let appointmentsByTimeslot: Appointment[][] = [];
        schedule.forEach(() => appointmentsByTimeslot.push([]));
        appointments.forEach((appt) =>
            appointmentsByTimeslot[appt.timeslot].push(appt),
        );

        // filter to only times with some appointments available,
        // or the first in a sequence of no availability, which will be rendered as a "gap"
        this.filteredSchedule = schedule = filterAppointmentsSchedule(schedule);

        let now = moment();

        // header row with times
        let headerRow = $('<tr></tr>').appendTo(this.appointmentsTableElem);
        this.headerElemsByTimeslot = {};
        let headerElems: JQuery[] = (this.headerElems = []);
        schedule.forEach((slots, i) => {
            let headerElem: JQuery;
            if (slots.numAvailable > 0) {
                headerElem = $(
                    `<th class="appointment-slots-header"><span>${slots.scheduledTime.format(
                        'h:mma',
                    )}</span></th>`,
                );
                if (slots.scheduledTime.format('h:mma').indexOf('00') !== -1) {
                    // on the hour
                    headerElem.addClass('appointment-slots-header-major');
                } else if (i === 0 || schedule[i - 1].numAvailable === 0) {
                    // first timeslot or first after a gap
                    headerElem.addClass('appointment-slots-header-major');
                } else {
                    // otherwise, it's minor
                    headerElem.addClass('appointment-slots-header-major');
                }

                let diff = slots.scheduledTime.diff(now, 'minutes');
                if (diff) {
                    // dim if appointment has recently passed
                    headerElem.addClass('appointment-slots-header-past');
                }
            } else {
                headerElem = $(
                    `<th class="appointment-slots-header"><span>&nbsp</span></th>`,
                );
            }
            headerElems.push(headerElem);
            this.headerElemsByTimeslot[slots.timeslot] = headerElem;
            headerRow.append(headerElem);
        });

        for (let r = 1; r <= maxAppts; ++r) {
            let row = $('<tr></tr>').appendTo(this.appointmentsTableElem);
            schedule.forEach((slots, i) => {
                let apptCell;
                let appts = appointmentsByTimeslot[slots.timeslot];
                if (slots.numAvailable < r) {
                    // no appointment slot
                    apptCell = $(
                        '<td class="appointment-cell appointment-cell-blank"><button type="button" class="btn btn-basic">&nbsp<br />&nbsp</button></td',
                    ).appendTo(row);
                } else if (appts.length < r) {
                    // unfilled
                    let appt = createAppointment(
                        {
                            id: 'unfilled',
                            queue: this.queue.page.queueId,
                            timeslot: slots.timeslot,
                            duration: slots.duration,
                            scheduledTime: moment(),
                        },
                        moment().tz('America/New_York').startOf('day'),
                    );
                    apptCell = $(
                        `<td class="appointment-cell"><span data-toggle="tooltip" data-html="true" title="Claim this slot!"><button type="button" class="btn btn-default">${slots.timeslot}<br />&nbsp</button></span></td>`,
                    ).appendTo(row);
                    apptCell.find('[data-toggle="tooltip"]').tooltip();
                    apptCell.find('button').click(() => {
                        this.appointmentViewer.setSelectedAppointment(appt);
                    });
                } else {
                    // scheduled appointment
                    let appt = appts[r - 1];
                    if (
                        appt.staffEmail !== undefined &&
                        appt.staffEmail === User.email()
                    ) {
                        let staffUniqname = appt.staffEmail.replace(
                            /@.*\..*/,
                            '',
                        );
                        apptCell = $(
                            `<td class="appointment-cell appointment-cell-claimed"><span data-toggle="tooltip" data-html="true" title="${
                                appt.name || '(no student)'
                            }<br />Click to show info below..."><button type="button" class="btn ${
                                appt.studentEmail === undefined
                                    ? 'btn-success'
                                    : 'btn-primary'
                            }">${
                                appt.name || '(no student)'
                            }<br /><span class="glyphicon glyphicon-flag"></span> ${staffUniqname}</button></span></td>`,
                        ).appendTo(row);
                    } else if (appt.staffEmail !== undefined) {
                        let staffUniqname = appt.staffEmail.replace(
                            /@.*\..*/,
                            '',
                        );
                        apptCell = $(
                            `<td class="appointment-cell"><span data-toggle="tooltip" data-html="true" title="${
                                appt.name || '(no student)'
                            }<br />Click to show info below..."><button type="button" class="btn btn-warning">${
                                appt.name || '(no student)'
                            }<br /><span class="glyphicon glyphicon-flag"></span> ${staffUniqname}</button></span></td>`,
                        ).appendTo(row);
                    } else {
                        apptCell = $(
                            `<td class="appointment-cell"><span data-toggle="tooltip" data-html="true" title="${appt.name}<br />Click to show info below..."><button type="button" class="btn btn-danger">${appt.name}<br />&nbsp</button></span></td>`,
                        ).appendTo(row);
                    }
                    apptCell.find('[data-toggle="tooltip"]').tooltip();
                    apptCell.find('button').click(() => {
                        this.appointmentViewer.setSelectedAppointment(appt);
                    });

                    // If we find a new appointment from the server with our selected id,
                    // go ahead and select it again to force an update with potential new data
                    if (appt.id === this.appointmentViewer.selected?.id) {
                        this.appointmentViewer.setSelectedAppointment(appt);
                        apptCell.addClass('appointment-cell-selected');
                    }

                    if (appt.scheduledTime.diff(now, 'minutes') < 2) {
                        // appointment is coming up

                        if (!this.notificationsGiven[appt.id]) {
                            if (appt.staffEmail === User.email()) {
                                // claimed by you
                                this.notificationsGiven[appt.id] = true;
                                let time = appt.scheduledTime.format('h:mma');
                                QueueApplication.instance.notify(
                                    `Your OH Appointment`,
                                    `You have an OH appointment for ${appt.name} at ${time}.`,
                                );
                            } else if (appt.staffEmail === '') {
                                // claimed by nobody
                                this.notificationsGiven[appt.id] = true;
                                let time = appt.scheduledTime.format('h:mma');
                                QueueApplication.instance.notify(
                                    `Unclaimed OH Appointment`,
                                    `Nobody has claimed the OH appointment for ${appt.name} at ${time}.`,
                                );
                            }
                            // else must be claimed by someone else
                        }
                    }
                }

                let diff = slots.scheduledTime.diff(now, 'minutes');
                if (diff < -slots.duration) {
                    // dim if appointment has passed
                    apptCell.addClass('appointment-cell-past');
                } else if (diff < 0) {
                    // dim if appointment has recently passed
                    apptCell.addClass('appointment-cell-recent-past');
                } else if (diff < 60) {
                    // appointments in the next hour should be wider
                    apptCell.addClass('appointment-cell-near-future');
                } else {
                    // all others
                    apptCell.addClass('appointment-cell-future');
                }
            });
        }

        if (firstTime) {
            this.scrollToNow(3000);
            this.crawlToNow(3000);
        }
    }
}

class StudentControls {
    private static _name = 'StudentControls';

    private queue: AppointmentsQueue;

    private elem: JQuery;
    private signUpForm: SignUpForm<true>;

    public readonly _act!: MessageResponses;

    constructor(queue: AppointmentsQueue, elem: JQuery) {
        this.queue = queue;
        this.elem = elem;

        let formElem = $('<div></div>').appendTo(this.elem);
        this.signUpForm = new SignUpForm(
            formElem,
            true,
            this.queue.page.mapImageSrc,
        );
        addListener(this.signUpForm, this);
    }

    public refreshSignUpEnabled() {
        var isEnabled = User.isUmich() && !this.queue.myRequest;
        this.signUpForm.setSignUpEnabled(isEnabled);
    }

    @messageResponse()
    private queueRefreshed() {
        this.refreshSignUpEnabled();
    }

    @messageResponse()
    private userSignedIn() {
        this.refreshSignUpEnabled();
    }

    public setMyAppointment() {
        this.signUpForm.setMyRequest(this.queue.myRequest);
    }

    public setAppointments(schedule: AppointmentSchedule) {
        this.signUpForm.setAppointments(schedule);
    }

    @messageResponse()
    private signUp(msg: Message<SignUpMessage>) {
        if (!this.queue.myRequest) {
            this.queue.signUp(
                msg.data.signUpName,
                msg.data.signUpLocation,
                msg.data.signUpDescription,
                msg.data.mapX,
                msg.data.mapY,
                msg.data.timeslot,
            );
        } else {
            this.queue.updateRequest(
                msg.data.signUpName,
                msg.data.signUpLocation,
                msg.data.signUpDescription,
                msg.data.mapX,
                msg.data.mapY,
                msg.data.timeslot,
            );
        }
    }

    @messageResponse()
    private removeRequest(msg: Message<Appointment>) {
        this.queue.removeAppointment(msg.data);
    }
}
function createAppointment(appData: any, startOfDay: moment.Moment) {
    return <Appointment>{
        kind: 'appointment',
        id: appData['id'],
        queueId: appData['queue'],
        timeslot: appData['timeslot'],
        duration: appData['duration'],
        scheduledTime: moment(appData['scheduled_time']),
        studentEmail: appData['student_email'],
        staffEmail: appData['staff_email'],
        name: appData['name'],
        location: appData['location'],
        description: appData['description'],
        mapX: appData['mapX'],
        mapY: appData['mapY'],
    };
}
