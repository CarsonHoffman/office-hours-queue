--
-- PostgreSQL database dump
--

-- Dumped from database version 12.3 (Debian 12.3-1.pgdg100+1)
-- Dumped by pg_dump version 12.3

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: announcements; Type: TABLE; Schema: public; Owner: queue
--

CREATE TABLE public.announcements (
    id character(27) NOT NULL COLLATE pg_catalog."C",
    queue character(27) NOT NULL,
    content text NOT NULL
);


ALTER TABLE public.announcements OWNER TO queue;

--
-- Name: appointment_schedules; Type: TABLE; Schema: public; Owner: queue
--

CREATE TABLE public.appointment_schedules (
    queue character(27) NOT NULL COLLATE pg_catalog."C",
    day smallint NOT NULL,
    duration bigint NOT NULL,
    padding bigint NOT NULL,
    schedule text NOT NULL
);


ALTER TABLE public.appointment_schedules OWNER TO queue;

--
-- Name: appointment_slots; Type: TABLE; Schema: public; Owner: queue
--

CREATE TABLE public.appointment_slots (
    id character(27) NOT NULL COLLATE pg_catalog."C",
    queue character(27) NOT NULL COLLATE pg_catalog."C",
    staff_email text,
    student_email text,
    scheduled_time timestamp with time zone NOT NULL,
    timeslot integer NOT NULL,
    duration integer NOT NULL,
    name text,
    location text,
    description text,
    map_x real,
    map_y real
);


ALTER TABLE public.appointment_slots OWNER TO queue;

--
-- Name: course_admins; Type: TABLE; Schema: public; Owner: queue
--

CREATE TABLE public.course_admins (
    course character(27) NOT NULL COLLATE pg_catalog."C",
    email text NOT NULL
);


ALTER TABLE public.course_admins OWNER TO queue;

--
-- Name: courses; Type: TABLE; Schema: public; Owner: queue
--

CREATE TABLE public.courses (
    id character(27) NOT NULL COLLATE pg_catalog."C",
    short_name text NOT NULL,
    full_name text NOT NULL
);


ALTER TABLE public.courses OWNER TO queue;

--
-- Name: groups; Type: TABLE; Schema: public; Owner: queue
--

CREATE TABLE public.groups (
    queue character(27) NOT NULL,
    email text NOT NULL,
    group_id character(27) NOT NULL
);


ALTER TABLE public.groups OWNER TO queue;


--
-- Name: messages; Type: TABLE; Schema: public; Owner: queue
--

CREATE TABLE public.messages (
    id character(27) NOT NULL,
    queue character(27) NOT NULL,
    content text NOT NULL,
    sender text NOT NULL,
    receiver text NOT NULL
);


ALTER TABLE public.messages OWNER TO queue;

--
-- Name: queue_entries; Type: TABLE; Schema: public; Owner: queue
--

CREATE TABLE public.queue_entries (
    id character(27) NOT NULL COLLATE pg_catalog."C",
    queue character(27) NOT NULL COLLATE pg_catalog."C",
    email text NOT NULL,
    name text NOT NULL,
    location text NOT NULL,
    map_x real NOT NULL,
    map_y real NOT NULL,
    description text NOT NULL,
    priority smallint NOT NULL,
    pinned boolean DEFAULT false NOT NULL,
    active boolean DEFAULT true, -- Nullable so that we can set up unique relation
    removed_by text,
    removed_at timestamp without time zone,
    helped boolean DEFAULT true NOT NULL
);


ALTER TABLE public.queue_entries OWNER TO queue;

ALTER TABLE ONLY public.queue_entries
    ADD CONSTRAINT one_active_entry_per_student_per_queue UNIQUE (queue, email, active);

--
-- Name: queues; Type: TABLE; Schema: public; Owner: queue
--

CREATE TABLE public.queues (
    id character(27) NOT NULL COLLATE pg_catalog."C",
    course character(27) NOT NULL COLLATE pg_catalog."C",
    location text NOT NULL,
    map text NOT NULL,
    active boolean NOT NULL,
    enable_location_field boolean DEFAULT true NOT NULL,
    prevent_unregistered boolean DEFAULT false NOT NULL,
    prevent_groups boolean DEFAULT false NOT NULL,
    prevent_groups_boost boolean DEFAULT false NOT NULL,
    prioritize_new boolean DEFAULT false NOT NULL,
    virtual boolean DEFAULT false NOT NULL,
    scheduled boolean DEFAULT false NOT NULL,
    manual_open boolean DEFAULT false NOT NULL,
    type text NOT NULL,
    name text NOT NULL
);


ALTER TABLE public.queues OWNER TO queue;

--
-- Name: roster; Type: TABLE; Schema: public; Owner: queue
--

CREATE TABLE public.roster (
    queue character(27) NOT NULL COLLATE pg_catalog."C",
    email text NOT NULL
);


ALTER TABLE public.roster OWNER TO queue;

--
-- Name: schedules; Type: TABLE; Schema: public; Owner: queue
--

CREATE TABLE public.schedules (
    queue character(27) NOT NULL COLLATE pg_catalog."C",
    day smallint NOT NULL,
    schedule character(48) NOT NULL
);


ALTER TABLE public.schedules OWNER TO queue;

--
-- Name: site_admins; Type: TABLE; Schema: public; Owner: queue
--

CREATE TABLE public.site_admins (
    email text NOT NULL
);


ALTER TABLE public.site_admins OWNER TO queue;
--
-- Name: teammates; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.teammates AS
 SELECT g2.queue,
    g1.email,
    g2.email AS teammate
   FROM (public.groups g1
     JOIN public.groups g2 ON (((g1.queue = g2.queue) AND (g1.group_id = g2.group_id) AND (g1.email <> g2.email))));


--
-- Name: announcements announcements_pkey; Type: CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.announcements
    ADD CONSTRAINT announcements_pkey PRIMARY KEY (id);


--
-- Name: appointment_schedules appointment_schedules_pkey; Type: CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.appointment_schedules
    ADD CONSTRAINT appointment_schedules_pkey PRIMARY KEY (queue, day);


--
-- Name: appointment_slots appointment_slots_pkey; Type: CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.appointment_slots
    ADD CONSTRAINT appointment_slots_pkey PRIMARY KEY (id);


--
-- Name: course_admins course_admins_course_email_key; Type: CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.course_admins
    ADD CONSTRAINT course_admins_pkey PRIMARY KEY (course, email);


--
-- Name: courses courses_pkey; Type: CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.courses
    ADD CONSTRAINT courses_pkey PRIMARY KEY (id);


--
-- Name: messages messages_pkey; Type: CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT messages_pkey PRIMARY KEY (id);


--
-- Name: groups one_group_per_student_per_queue; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.groups
    ADD CONSTRAINT one_group_per_student_per_queue UNIQUE (queue, email);


--
-- Name: queue_entries queueentries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.queue_entries
    ADD CONSTRAINT queueentries_pkey PRIMARY KEY (id);


--
-- Name: queues queues_pkey; Type: CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.queues
    ADD CONSTRAINT queues_pkey PRIMARY KEY (id);


--
-- Name: roster roster_queue_email_key; Type: CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.roster
    ADD CONSTRAINT roster_pkey PRIMARY KEY (queue, email);


--
-- Name: schedules schedules_queue_day_key; Type: CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.schedules
    ADD CONSTRAINT schedules_pkey PRIMARY KEY (queue, day);


--
-- Name: site_admins site_admins_pkey; Type: CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.site_admins
    ADD CONSTRAINT site_admins_pkey PRIMARY KEY (email);


--
-- Name: queue_entries_queue_idx; Type: INDEX; Schema: public; Owner: queue
--

CREATE INDEX queue_entries_queue_idx ON public.queue_entries USING btree (queue);


--
-- Name: queue_entries_queue_removed_idx; Type: INDEX; Schema: public; Owner: queue
--

CREATE INDEX queue_entries_queue_removed_idx ON public.queue_entries USING btree (queue, removed);


--
-- Name: queue_entries_queue_removed_removed_at_idx; Type: INDEX; Schema: public; Owner: queue
--

CREATE INDEX queue_entries_queue_removed_removed_at_idx ON public.queue_entries USING btree (queue, removed, removed_at);


--
-- Name: announcements announcements_queue_fkey; Type: FK CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.announcements
    ADD CONSTRAINT announcements_queue_fkey FOREIGN KEY (queue) REFERENCES public.queues(id) ON DELETE CASCADE;


--
-- Name: appointment_schedules appointment_schedules_queue_fkey; Type: FK CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.appointment_schedules
    ADD CONSTRAINT appointment_schedules_queue_fkey FOREIGN KEY (queue) REFERENCES public.queues(id) ON DELETE CASCADE;


--
-- Name: appointment_slots appointment_slots_queue_fkey; Type: FK CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.appointment_slots
    ADD CONSTRAINT appointment_slots_queue_fkey FOREIGN KEY (queue) REFERENCES public.queues(id) ON DELETE CASCADE;


--
-- Name: course_admins course_admins_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.course_admins
    ADD CONSTRAINT course_admins_course_fkey FOREIGN KEY (course) REFERENCES public.courses(id) ON DELETE CASCADE;


--
-- Name: groups groups_queue_fkey; Type: FK CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.groups
    ADD CONSTRAINT groups_queue_fkey FOREIGN KEY (queue) REFERENCES public.queues(id) ON DELETE CASCADE;


--
-- Name: messages messages_queue_fkey; Type: FK CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT messages_queue_fkey FOREIGN KEY (queue) REFERENCES public.queues(id) ON DELETE CASCADE;


--
-- Name: queue_entries queueentries_queue_fkey; Type: FK CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.queue_entries
    ADD CONSTRAINT queueentries_queue_fkey FOREIGN KEY (queue) REFERENCES public.queues(id) ON DELETE CASCADE;


--
-- Name: queues queues_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.queues
    ADD CONSTRAINT queues_course_fkey FOREIGN KEY (course) REFERENCES public.courses(id) ON DELETE CASCADE;


--
-- Name: roster roster_queue_fkey; Type: FK CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.roster
    ADD CONSTRAINT roster_queue_fkey FOREIGN KEY (queue) REFERENCES public.queues(id) ON DELETE CASCADE;


--
-- Name: schedules schedules_queue_fkey; Type: FK CONSTRAINT; Schema: public; Owner: queue
--

ALTER TABLE ONLY public.schedules
    ADD CONSTRAINT schedules_queue_fkey FOREIGN KEY (queue) REFERENCES public.queues(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

