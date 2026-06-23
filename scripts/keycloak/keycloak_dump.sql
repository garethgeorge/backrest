--
-- PostgreSQL database dump
--

\restrict 20yC9tXS2fLtRIfQaX1XoyQvrCv3DQIgGdME3VPjlZdRoy42LNnXP26K2ctb9Cy

-- Dumped from database version 18.4 (Debian 18.4-1.pgdg13+1)
-- Dumped by pg_dump version 18.4 (Debian 18.4-1.pgdg13+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
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
-- Name: admin_event_entity; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.admin_event_entity (
    id character varying(36) NOT NULL,
    admin_event_time bigint,
    realm_id character varying(255),
    operation_type character varying(255),
    auth_realm_id character varying(255),
    auth_client_id character varying(255),
    auth_user_id character varying(255),
    ip_address character varying(255),
    resource_path character varying(2550),
    representation text,
    error character varying(255),
    resource_type character varying(64),
    details_json text
);


ALTER TABLE public.admin_event_entity OWNER TO keycloak;

--
-- Name: associated_policy; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.associated_policy (
    policy_id character varying(36) NOT NULL,
    associated_policy_id character varying(36) NOT NULL
);


ALTER TABLE public.associated_policy OWNER TO keycloak;

--
-- Name: authentication_execution; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.authentication_execution (
    id character varying(36) NOT NULL,
    alias character varying(255),
    authenticator character varying(36),
    realm_id character varying(36),
    flow_id character varying(36),
    requirement integer,
    priority integer,
    authenticator_flow boolean DEFAULT false NOT NULL,
    auth_flow_id character varying(36),
    auth_config character varying(36)
);


ALTER TABLE public.authentication_execution OWNER TO keycloak;

--
-- Name: authentication_flow; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.authentication_flow (
    id character varying(36) NOT NULL,
    alias character varying(255),
    description character varying(255),
    realm_id character varying(36),
    provider_id character varying(36) DEFAULT 'basic-flow'::character varying NOT NULL,
    top_level boolean DEFAULT false NOT NULL,
    built_in boolean DEFAULT false NOT NULL
);


ALTER TABLE public.authentication_flow OWNER TO keycloak;

--
-- Name: authenticator_config; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.authenticator_config (
    id character varying(36) CONSTRAINT authenticator_id_not_null NOT NULL,
    alias character varying(255),
    realm_id character varying(36)
);


ALTER TABLE public.authenticator_config OWNER TO keycloak;

--
-- Name: authenticator_config_entry; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.authenticator_config_entry (
    authenticator_id character varying(36) CONSTRAINT authenticator_config_authenticator_id_not_null NOT NULL,
    value text,
    name character varying(255) CONSTRAINT authenticator_config_name_not_null NOT NULL
);


ALTER TABLE public.authenticator_config_entry OWNER TO keycloak;

--
-- Name: broker_link; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.broker_link (
    identity_provider character varying(255) NOT NULL,
    storage_provider_id character varying(255),
    realm_id character varying(36) NOT NULL,
    broker_user_id character varying(255),
    broker_username character varying(255),
    token text,
    user_id character varying(255) NOT NULL
);


ALTER TABLE public.broker_link OWNER TO keycloak;

--
-- Name: client; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.client (
    id character varying(36) NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    full_scope_allowed boolean DEFAULT false NOT NULL,
    client_id character varying(255),
    not_before integer,
    public_client boolean DEFAULT false NOT NULL,
    secret character varying(255),
    base_url character varying(255),
    bearer_only boolean DEFAULT false NOT NULL,
    management_url character varying(255),
    surrogate_auth_required boolean DEFAULT false NOT NULL,
    realm_id character varying(36),
    protocol character varying(255),
    node_rereg_timeout integer DEFAULT 0,
    frontchannel_logout boolean DEFAULT false NOT NULL,
    consent_required boolean DEFAULT false NOT NULL,
    name character varying(255),
    service_accounts_enabled boolean DEFAULT false NOT NULL,
    client_authenticator_type character varying(255),
    root_url character varying(255),
    description character varying(255),
    registration_token character varying(255),
    standard_flow_enabled boolean DEFAULT true NOT NULL,
    implicit_flow_enabled boolean DEFAULT false NOT NULL,
    direct_access_grants_enabled boolean DEFAULT false NOT NULL,
    always_display_in_console boolean DEFAULT false NOT NULL
);


ALTER TABLE public.client OWNER TO keycloak;

--
-- Name: client_attributes; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.client_attributes (
    client_id character varying(36) NOT NULL,
    name character varying(255) NOT NULL,
    value text
);


ALTER TABLE public.client_attributes OWNER TO keycloak;

--
-- Name: client_auth_flow_bindings; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.client_auth_flow_bindings (
    client_id character varying(36) NOT NULL,
    flow_id character varying(36),
    binding_name character varying(255) NOT NULL
);


ALTER TABLE public.client_auth_flow_bindings OWNER TO keycloak;

--
-- Name: client_initial_access; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.client_initial_access (
    id character varying(36) NOT NULL,
    realm_id character varying(36) NOT NULL,
    "timestamp" integer,
    expiration integer,
    count integer,
    remaining_count integer
);


ALTER TABLE public.client_initial_access OWNER TO keycloak;

--
-- Name: client_node_registrations; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.client_node_registrations (
    client_id character varying(36) CONSTRAINT app_node_registrations_application_id_not_null NOT NULL,
    value integer,
    name character varying(255) CONSTRAINT app_node_registrations_name_not_null NOT NULL
);


ALTER TABLE public.client_node_registrations OWNER TO keycloak;

--
-- Name: client_scope; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.client_scope (
    id character varying(36) CONSTRAINT client_template_id_not_null NOT NULL,
    name character varying(255),
    realm_id character varying(36),
    description character varying(255),
    protocol character varying(255)
);


ALTER TABLE public.client_scope OWNER TO keycloak;

--
-- Name: client_scope_attributes; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.client_scope_attributes (
    scope_id character varying(36) CONSTRAINT client_template_attributes_template_id_not_null NOT NULL,
    value character varying(2048),
    name character varying(255) CONSTRAINT client_template_attributes_name_not_null NOT NULL
);


ALTER TABLE public.client_scope_attributes OWNER TO keycloak;

--
-- Name: client_scope_client; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.client_scope_client (
    client_id character varying(255) NOT NULL,
    scope_id character varying(255) NOT NULL,
    default_scope boolean DEFAULT false NOT NULL
);


ALTER TABLE public.client_scope_client OWNER TO keycloak;

--
-- Name: client_scope_role_mapping; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.client_scope_role_mapping (
    scope_id character varying(36) CONSTRAINT template_scope_mapping_template_id_not_null NOT NULL,
    role_id character varying(36) CONSTRAINT template_scope_mapping_role_id_not_null NOT NULL
);


ALTER TABLE public.client_scope_role_mapping OWNER TO keycloak;

--
-- Name: component; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.component (
    id character varying(36) NOT NULL,
    name character varying(255),
    parent_id character varying(36),
    provider_id character varying(36),
    provider_type character varying(255),
    realm_id character varying(36),
    sub_type character varying(255)
);


ALTER TABLE public.component OWNER TO keycloak;

--
-- Name: component_config; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.component_config (
    id character varying(36) NOT NULL,
    component_id character varying(36) NOT NULL,
    name character varying(255) NOT NULL,
    value text
);


ALTER TABLE public.component_config OWNER TO keycloak;

--
-- Name: composite_role; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.composite_role (
    composite character varying(36) NOT NULL,
    child_role character varying(36) NOT NULL
);


ALTER TABLE public.composite_role OWNER TO keycloak;

--
-- Name: credential; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.credential (
    id character varying(36) NOT NULL,
    salt bytea,
    type character varying(255),
    user_id character varying(36),
    created_date bigint,
    user_label character varying(255),
    secret_data text,
    credential_data text,
    priority integer,
    version integer DEFAULT 0
);


ALTER TABLE public.credential OWNER TO keycloak;

--
-- Name: databasechangelog; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.databasechangelog (
    id character varying(255) NOT NULL,
    author character varying(255) NOT NULL,
    filename character varying(255) NOT NULL,
    dateexecuted timestamp without time zone NOT NULL,
    orderexecuted integer NOT NULL,
    exectype character varying(10) NOT NULL,
    md5sum character varying(35),
    description character varying(255),
    comments character varying(255),
    tag character varying(255),
    liquibase character varying(20),
    contexts character varying(255),
    labels character varying(255),
    deployment_id character varying(10)
);


ALTER TABLE public.databasechangelog OWNER TO keycloak;

--
-- Name: databasechangeloglock; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.databasechangeloglock (
    id integer NOT NULL,
    locked boolean NOT NULL,
    lockgranted timestamp without time zone,
    lockedby character varying(255)
);


ALTER TABLE public.databasechangeloglock OWNER TO keycloak;

--
-- Name: default_client_scope; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.default_client_scope (
    realm_id character varying(36) NOT NULL,
    scope_id character varying(36) NOT NULL,
    default_scope boolean DEFAULT false NOT NULL
);


ALTER TABLE public.default_client_scope OWNER TO keycloak;

--
-- Name: event_entity; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.event_entity (
    id character varying(36) NOT NULL,
    client_id character varying(255),
    details_json character varying(2550),
    error character varying(255),
    ip_address character varying(255),
    realm_id character varying(255),
    session_id character varying(255),
    event_time bigint,
    type character varying(255),
    user_id character varying(255),
    details_json_long_value text
);


ALTER TABLE public.event_entity OWNER TO keycloak;

--
-- Name: fed_user_attribute; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.fed_user_attribute (
    id character varying(36) NOT NULL,
    name character varying(255) NOT NULL,
    user_id character varying(255) NOT NULL,
    realm_id character varying(36) NOT NULL,
    storage_provider_id character varying(36),
    value character varying(2024),
    long_value_hash bytea,
    long_value_hash_lower_case bytea,
    long_value text
);


ALTER TABLE public.fed_user_attribute OWNER TO keycloak;

--
-- Name: fed_user_consent; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.fed_user_consent (
    id character varying(36) NOT NULL,
    client_id character varying(255),
    user_id character varying(255) NOT NULL,
    realm_id character varying(36) NOT NULL,
    storage_provider_id character varying(36),
    created_date bigint,
    last_updated_date bigint,
    client_storage_provider character varying(36),
    external_client_id character varying(255)
);


ALTER TABLE public.fed_user_consent OWNER TO keycloak;

--
-- Name: fed_user_consent_cl_scope; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.fed_user_consent_cl_scope (
    user_consent_id character varying(36) NOT NULL,
    scope_id character varying(36) NOT NULL
);


ALTER TABLE public.fed_user_consent_cl_scope OWNER TO keycloak;

--
-- Name: fed_user_credential; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.fed_user_credential (
    id character varying(36) NOT NULL,
    salt bytea,
    type character varying(255),
    created_date bigint,
    user_id character varying(255) NOT NULL,
    realm_id character varying(36) NOT NULL,
    storage_provider_id character varying(36),
    user_label character varying(255),
    secret_data text,
    credential_data text,
    priority integer
);


ALTER TABLE public.fed_user_credential OWNER TO keycloak;

--
-- Name: fed_user_group_membership; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.fed_user_group_membership (
    group_id character varying(36) NOT NULL,
    user_id character varying(255) NOT NULL,
    realm_id character varying(36) NOT NULL,
    storage_provider_id character varying(36)
);


ALTER TABLE public.fed_user_group_membership OWNER TO keycloak;

--
-- Name: fed_user_required_action; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.fed_user_required_action (
    required_action character varying(255) DEFAULT ' '::character varying NOT NULL,
    user_id character varying(255) NOT NULL,
    realm_id character varying(36) NOT NULL,
    storage_provider_id character varying(36)
);


ALTER TABLE public.fed_user_required_action OWNER TO keycloak;

--
-- Name: fed_user_role_mapping; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.fed_user_role_mapping (
    role_id character varying(36) NOT NULL,
    user_id character varying(255) NOT NULL,
    realm_id character varying(36) NOT NULL,
    storage_provider_id character varying(36)
);


ALTER TABLE public.fed_user_role_mapping OWNER TO keycloak;

--
-- Name: federated_identity; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.federated_identity (
    identity_provider character varying(255) NOT NULL,
    realm_id character varying(36),
    federated_user_id character varying(255),
    federated_username character varying(255),
    token text,
    user_id character varying(36) NOT NULL
);


ALTER TABLE public.federated_identity OWNER TO keycloak;

--
-- Name: federated_user; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.federated_user (
    id character varying(255) NOT NULL,
    storage_provider_id character varying(255),
    realm_id character varying(36) NOT NULL
);


ALTER TABLE public.federated_user OWNER TO keycloak;

--
-- Name: group_attribute; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.group_attribute (
    id character varying(36) DEFAULT 'sybase-needs-something-here'::character varying NOT NULL,
    name character varying(255) NOT NULL,
    value character varying(255),
    group_id character varying(36) NOT NULL
);


ALTER TABLE public.group_attribute OWNER TO keycloak;

--
-- Name: group_role_mapping; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.group_role_mapping (
    role_id character varying(36) NOT NULL,
    group_id character varying(36) NOT NULL
);


ALTER TABLE public.group_role_mapping OWNER TO keycloak;

--
-- Name: identity_provider; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.identity_provider (
    internal_id character varying(36) NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    provider_alias character varying(255),
    provider_id character varying(255),
    store_token boolean,
    authenticate_by_default boolean,
    realm_id character varying(36),
    add_token_role boolean,
    trust_email boolean,
    first_broker_login_flow_id character varying(36),
    post_broker_login_flow_id character varying(36),
    provider_display_name character varying(255),
    link_only boolean,
    organization_id character varying(255),
    hide_on_login boolean
);


ALTER TABLE public.identity_provider OWNER TO keycloak;

--
-- Name: identity_provider_config; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.identity_provider_config (
    identity_provider_id character varying(36) NOT NULL,
    value text,
    name character varying(255) NOT NULL
);


ALTER TABLE public.identity_provider_config OWNER TO keycloak;

--
-- Name: identity_provider_mapper; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.identity_provider_mapper (
    id character varying(36) NOT NULL,
    name character varying(255) NOT NULL,
    idp_alias character varying(255) NOT NULL,
    idp_mapper_name character varying(255) NOT NULL,
    realm_id character varying(36) NOT NULL
);


ALTER TABLE public.identity_provider_mapper OWNER TO keycloak;

--
-- Name: idp_mapper_config; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.idp_mapper_config (
    idp_mapper_id character varying(36) NOT NULL,
    value text,
    name character varying(255) NOT NULL
);


ALTER TABLE public.idp_mapper_config OWNER TO keycloak;

--
-- Name: jgroups_ping; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.jgroups_ping (
    address character varying(200) NOT NULL,
    name character varying(200),
    cluster_name character varying(200) NOT NULL,
    ip character varying(200) NOT NULL,
    coord boolean
);


ALTER TABLE public.jgroups_ping OWNER TO keycloak;

--
-- Name: keycloak_group; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.keycloak_group (
    id character varying(36) NOT NULL,
    name character varying(255),
    parent_group character varying(36) NOT NULL,
    realm_id character varying(36),
    type integer DEFAULT 0 NOT NULL,
    description character varying(255),
    org_id character varying(255),
    created_timestamp bigint,
    last_modified_timestamp bigint
);


ALTER TABLE public.keycloak_group OWNER TO keycloak;

--
-- Name: keycloak_role; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.keycloak_role (
    id character varying(36) NOT NULL,
    client_realm_constraint character varying(255),
    client_role boolean DEFAULT false CONSTRAINT keycloak_role_application_role_not_null NOT NULL,
    description character varying(255),
    name character varying(255),
    realm_id character varying(255),
    client character varying(36),
    realm character varying(36)
);


ALTER TABLE public.keycloak_role OWNER TO keycloak;

--
-- Name: migration_model; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.migration_model (
    id character varying(36) NOT NULL,
    version character varying(36),
    update_time bigint DEFAULT 0 NOT NULL
);


ALTER TABLE public.migration_model OWNER TO keycloak;

--
-- Name: offline_client_session; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.offline_client_session (
    user_session_id character varying(36) NOT NULL,
    client_id character varying(255) NOT NULL,
    offline_flag character varying(4) NOT NULL,
    "timestamp" integer,
    data text,
    client_storage_provider character varying(36) DEFAULT 'local'::character varying NOT NULL,
    external_client_id character varying(255) DEFAULT 'local'::character varying NOT NULL,
    version integer DEFAULT 0,
    realm_id character varying(36)
);


ALTER TABLE public.offline_client_session OWNER TO keycloak;

--
-- Name: offline_user_session; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.offline_user_session (
    user_session_id character varying(36) NOT NULL,
    user_id character varying(255) NOT NULL,
    realm_id character varying(36) NOT NULL,
    created_on integer NOT NULL,
    offline_flag character varying(4) NOT NULL,
    data text,
    last_session_refresh integer DEFAULT 0 NOT NULL,
    broker_session_id character varying(1024),
    version integer DEFAULT 0,
    remember_me boolean
);


ALTER TABLE public.offline_user_session OWNER TO keycloak;

--
-- Name: org; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.org (
    id character varying(255) NOT NULL,
    enabled boolean NOT NULL,
    realm_id character varying(255) NOT NULL,
    group_id character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    description character varying(4000),
    alias character varying(255) NOT NULL,
    redirect_url character varying(2048)
);


ALTER TABLE public.org OWNER TO keycloak;

--
-- Name: org_domain; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.org_domain (
    id character varying(36) NOT NULL,
    name character varying(255) NOT NULL,
    verified boolean NOT NULL,
    org_id character varying(255) NOT NULL
);


ALTER TABLE public.org_domain OWNER TO keycloak;

--
-- Name: org_invitation; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.org_invitation (
    id character varying(36) NOT NULL,
    organization_id character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    first_name character varying(255),
    last_name character varying(255),
    created_at integer NOT NULL,
    expires_at integer,
    invite_link character varying(2048)
);


ALTER TABLE public.org_invitation OWNER TO keycloak;

--
-- Name: policy_config; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.policy_config (
    policy_id character varying(36) NOT NULL,
    name character varying(255) NOT NULL,
    value text
);


ALTER TABLE public.policy_config OWNER TO keycloak;

--
-- Name: protocol_mapper; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.protocol_mapper (
    id character varying(36) NOT NULL,
    name character varying(255) NOT NULL,
    protocol character varying(255) NOT NULL,
    protocol_mapper_name character varying(255) NOT NULL,
    client_id character varying(36),
    client_scope_id character varying(36)
);


ALTER TABLE public.protocol_mapper OWNER TO keycloak;

--
-- Name: protocol_mapper_config; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.protocol_mapper_config (
    protocol_mapper_id character varying(36) NOT NULL,
    value text,
    name character varying(255) NOT NULL
);


ALTER TABLE public.protocol_mapper_config OWNER TO keycloak;

--
-- Name: realm; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.realm (
    id character varying(36) NOT NULL,
    access_code_lifespan integer,
    user_action_lifespan integer,
    access_token_lifespan integer,
    account_theme character varying(255),
    admin_theme character varying(255),
    email_theme character varying(255),
    enabled boolean DEFAULT false NOT NULL,
    events_enabled boolean DEFAULT false NOT NULL,
    events_expiration bigint,
    login_theme character varying(255),
    name character varying(255),
    not_before integer,
    password_policy character varying(2550),
    registration_allowed boolean DEFAULT false NOT NULL,
    remember_me boolean DEFAULT false NOT NULL,
    reset_password_allowed boolean DEFAULT false NOT NULL,
    social boolean DEFAULT false NOT NULL,
    ssl_required character varying(255),
    sso_idle_timeout integer,
    sso_max_lifespan integer,
    update_profile_on_soc_login boolean DEFAULT false NOT NULL,
    verify_email boolean DEFAULT false NOT NULL,
    master_admin_client character varying(36),
    login_lifespan integer,
    internationalization_enabled boolean DEFAULT false NOT NULL,
    default_locale character varying(255),
    reg_email_as_username boolean DEFAULT false NOT NULL,
    admin_events_enabled boolean DEFAULT false NOT NULL,
    admin_events_details_enabled boolean DEFAULT false NOT NULL,
    edit_username_allowed boolean DEFAULT false NOT NULL,
    otp_policy_counter integer DEFAULT 0,
    otp_policy_window integer DEFAULT 1,
    otp_policy_period integer DEFAULT 30,
    otp_policy_digits integer DEFAULT 6,
    otp_policy_alg character varying(36) DEFAULT 'HmacSHA1'::character varying,
    otp_policy_type character varying(36) DEFAULT 'totp'::character varying,
    browser_flow character varying(36),
    registration_flow character varying(36),
    direct_grant_flow character varying(36),
    reset_credentials_flow character varying(36),
    client_auth_flow character varying(36),
    offline_session_idle_timeout integer DEFAULT 0,
    revoke_refresh_token boolean DEFAULT false NOT NULL,
    access_token_life_implicit integer DEFAULT 0,
    login_with_email_allowed boolean DEFAULT true NOT NULL,
    duplicate_emails_allowed boolean DEFAULT false NOT NULL,
    docker_auth_flow character varying(36),
    refresh_token_max_reuse integer DEFAULT 0,
    allow_user_managed_access boolean DEFAULT false NOT NULL,
    sso_max_lifespan_remember_me integer DEFAULT 0 NOT NULL,
    sso_idle_timeout_remember_me integer DEFAULT 0 NOT NULL,
    default_role character varying(255)
);


ALTER TABLE public.realm OWNER TO keycloak;

--
-- Name: realm_attribute; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.realm_attribute (
    name character varying(255) NOT NULL,
    realm_id character varying(36) NOT NULL,
    value text
);


ALTER TABLE public.realm_attribute OWNER TO keycloak;

--
-- Name: realm_default_groups; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.realm_default_groups (
    realm_id character varying(36) NOT NULL,
    group_id character varying(36) NOT NULL
);


ALTER TABLE public.realm_default_groups OWNER TO keycloak;

--
-- Name: realm_enabled_event_types; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.realm_enabled_event_types (
    realm_id character varying(36) NOT NULL,
    value character varying(255) NOT NULL
);


ALTER TABLE public.realm_enabled_event_types OWNER TO keycloak;

--
-- Name: realm_events_listeners; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.realm_events_listeners (
    realm_id character varying(36) NOT NULL,
    value character varying(255) NOT NULL
);


ALTER TABLE public.realm_events_listeners OWNER TO keycloak;

--
-- Name: realm_localizations; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.realm_localizations (
    realm_id character varying(255) NOT NULL,
    locale character varying(255) NOT NULL,
    texts text NOT NULL
);


ALTER TABLE public.realm_localizations OWNER TO keycloak;

--
-- Name: realm_required_credential; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.realm_required_credential (
    type character varying(255) NOT NULL,
    form_label character varying(255),
    input boolean DEFAULT false NOT NULL,
    secret boolean DEFAULT false NOT NULL,
    realm_id character varying(36) NOT NULL
);


ALTER TABLE public.realm_required_credential OWNER TO keycloak;

--
-- Name: realm_smtp_config; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.realm_smtp_config (
    realm_id character varying(36) NOT NULL,
    value character varying(255),
    name character varying(255) NOT NULL
);


ALTER TABLE public.realm_smtp_config OWNER TO keycloak;

--
-- Name: realm_supported_locales; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.realm_supported_locales (
    realm_id character varying(36) NOT NULL,
    value character varying(255) NOT NULL
);


ALTER TABLE public.realm_supported_locales OWNER TO keycloak;

--
-- Name: redirect_uris; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.redirect_uris (
    client_id character varying(36) NOT NULL,
    value character varying(255) NOT NULL
);


ALTER TABLE public.redirect_uris OWNER TO keycloak;

--
-- Name: required_action_config; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.required_action_config (
    required_action_id character varying(36) NOT NULL,
    value text,
    name character varying(255) NOT NULL
);


ALTER TABLE public.required_action_config OWNER TO keycloak;

--
-- Name: required_action_provider; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.required_action_provider (
    id character varying(36) NOT NULL,
    alias character varying(255),
    name character varying(255),
    realm_id character varying(36),
    enabled boolean DEFAULT false NOT NULL,
    default_action boolean DEFAULT false NOT NULL,
    provider_id character varying(255),
    priority integer
);


ALTER TABLE public.required_action_provider OWNER TO keycloak;

--
-- Name: resource_attribute; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.resource_attribute (
    id character varying(36) DEFAULT 'sybase-needs-something-here'::character varying NOT NULL,
    name character varying(255) NOT NULL,
    value character varying(255),
    resource_id character varying(36) NOT NULL
);


ALTER TABLE public.resource_attribute OWNER TO keycloak;

--
-- Name: resource_policy; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.resource_policy (
    resource_id character varying(36) NOT NULL,
    policy_id character varying(36) NOT NULL
);


ALTER TABLE public.resource_policy OWNER TO keycloak;

--
-- Name: resource_scope; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.resource_scope (
    resource_id character varying(36) NOT NULL,
    scope_id character varying(36) NOT NULL
);


ALTER TABLE public.resource_scope OWNER TO keycloak;

--
-- Name: resource_server; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.resource_server (
    id character varying(36) CONSTRAINT resource_server_client_id_not_null NOT NULL,
    allow_rs_remote_mgmt boolean DEFAULT false NOT NULL,
    policy_enforce_mode smallint NOT NULL,
    decision_strategy smallint DEFAULT 1 NOT NULL
);


ALTER TABLE public.resource_server OWNER TO keycloak;

--
-- Name: resource_server_perm_ticket; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.resource_server_perm_ticket (
    id character varying(36) NOT NULL,
    owner character varying(255) NOT NULL,
    requester character varying(255) NOT NULL,
    created_timestamp bigint NOT NULL,
    granted_timestamp bigint,
    resource_id character varying(36) NOT NULL,
    scope_id character varying(36),
    resource_server_id character varying(36) NOT NULL,
    policy_id character varying(36)
);


ALTER TABLE public.resource_server_perm_ticket OWNER TO keycloak;

--
-- Name: resource_server_policy; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.resource_server_policy (
    id character varying(36) NOT NULL,
    name character varying(255) NOT NULL,
    description character varying(255),
    type character varying(255) NOT NULL,
    decision_strategy smallint,
    logic smallint,
    resource_server_id character varying(36) CONSTRAINT resource_server_policy_resource_server_client_id_not_null NOT NULL,
    owner character varying(255)
);


ALTER TABLE public.resource_server_policy OWNER TO keycloak;

--
-- Name: resource_server_resource; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.resource_server_resource (
    id character varying(36) NOT NULL,
    name character varying(255) NOT NULL,
    type character varying(255),
    icon_uri character varying(255),
    owner character varying(255) NOT NULL,
    resource_server_id character varying(36) CONSTRAINT resource_server_resource_resource_server_client_id_not_null NOT NULL,
    owner_managed_access boolean DEFAULT false NOT NULL,
    display_name character varying(255)
);


ALTER TABLE public.resource_server_resource OWNER TO keycloak;

--
-- Name: resource_server_scope; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.resource_server_scope (
    id character varying(36) NOT NULL,
    name character varying(255) NOT NULL,
    icon_uri character varying(255),
    resource_server_id character varying(36) CONSTRAINT resource_server_scope_resource_server_client_id_not_null NOT NULL,
    display_name character varying(255)
);


ALTER TABLE public.resource_server_scope OWNER TO keycloak;

--
-- Name: resource_uris; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.resource_uris (
    resource_id character varying(36) NOT NULL,
    value character varying(255) NOT NULL
);


ALTER TABLE public.resource_uris OWNER TO keycloak;

--
-- Name: revoked_token; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.revoked_token (
    id character varying(255) NOT NULL,
    expire bigint NOT NULL
);


ALTER TABLE public.revoked_token OWNER TO keycloak;

--
-- Name: role_attribute; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.role_attribute (
    id character varying(36) NOT NULL,
    role_id character varying(36) NOT NULL,
    name character varying(255) NOT NULL,
    value character varying(255)
);


ALTER TABLE public.role_attribute OWNER TO keycloak;

--
-- Name: scope_mapping; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.scope_mapping (
    client_id character varying(36) NOT NULL,
    role_id character varying(36) NOT NULL
);


ALTER TABLE public.scope_mapping OWNER TO keycloak;

--
-- Name: scope_policy; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.scope_policy (
    scope_id character varying(36) NOT NULL,
    policy_id character varying(36) NOT NULL
);


ALTER TABLE public.scope_policy OWNER TO keycloak;

--
-- Name: server_config; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.server_config (
    server_config_key character varying(255) NOT NULL,
    value text NOT NULL,
    version integer DEFAULT 0
);


ALTER TABLE public.server_config OWNER TO keycloak;

--
-- Name: user_attribute; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.user_attribute (
    name character varying(255) NOT NULL,
    value character varying(255),
    user_id character varying(36) NOT NULL,
    id character varying(36) DEFAULT 'sybase-needs-something-here'::character varying NOT NULL,
    long_value_hash bytea,
    long_value_hash_lower_case bytea,
    long_value text
);


ALTER TABLE public.user_attribute OWNER TO keycloak;

--
-- Name: user_consent; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.user_consent (
    id character varying(36) NOT NULL,
    client_id character varying(255),
    user_id character varying(36) NOT NULL,
    created_date bigint,
    last_updated_date bigint,
    client_storage_provider character varying(36),
    external_client_id character varying(255)
);


ALTER TABLE public.user_consent OWNER TO keycloak;

--
-- Name: user_consent_client_scope; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.user_consent_client_scope (
    user_consent_id character varying(36) NOT NULL,
    scope_id character varying(36) NOT NULL
);


ALTER TABLE public.user_consent_client_scope OWNER TO keycloak;

--
-- Name: user_entity; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.user_entity (
    id character varying(36) NOT NULL,
    email character varying(255),
    email_constraint character varying(255),
    email_verified boolean DEFAULT false NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    federation_link character varying(255),
    first_name character varying(255),
    last_name character varying(255),
    realm_id character varying(255),
    username character varying(255),
    created_timestamp bigint,
    service_account_client_link character varying(255),
    not_before integer DEFAULT 0 NOT NULL,
    last_modified_timestamp bigint
);


ALTER TABLE public.user_entity OWNER TO keycloak;

--
-- Name: user_federation_config; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.user_federation_config (
    user_federation_provider_id character varying(36) NOT NULL,
    value character varying(255),
    name character varying(255) NOT NULL
);


ALTER TABLE public.user_federation_config OWNER TO keycloak;

--
-- Name: user_federation_mapper; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.user_federation_mapper (
    id character varying(36) NOT NULL,
    name character varying(255) NOT NULL,
    federation_provider_id character varying(36) NOT NULL,
    federation_mapper_type character varying(255) NOT NULL,
    realm_id character varying(36) NOT NULL
);


ALTER TABLE public.user_federation_mapper OWNER TO keycloak;

--
-- Name: user_federation_mapper_config; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.user_federation_mapper_config (
    user_federation_mapper_id character varying(36) CONSTRAINT user_federation_mapper_confi_user_federation_mapper_id_not_null NOT NULL,
    value character varying(255),
    name character varying(255) NOT NULL
);


ALTER TABLE public.user_federation_mapper_config OWNER TO keycloak;

--
-- Name: user_federation_provider; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.user_federation_provider (
    id character varying(36) NOT NULL,
    changed_sync_period integer,
    display_name character varying(255),
    full_sync_period integer,
    last_sync integer,
    priority integer,
    provider_name character varying(255),
    realm_id character varying(36)
);


ALTER TABLE public.user_federation_provider OWNER TO keycloak;

--
-- Name: user_group_membership; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.user_group_membership (
    group_id character varying(36) NOT NULL,
    user_id character varying(36) NOT NULL,
    membership_type character varying(255) NOT NULL
);


ALTER TABLE public.user_group_membership OWNER TO keycloak;

--
-- Name: user_required_action; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.user_required_action (
    user_id character varying(36) NOT NULL,
    required_action character varying(255) DEFAULT ' '::character varying NOT NULL
);


ALTER TABLE public.user_required_action OWNER TO keycloak;

--
-- Name: user_role_mapping; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.user_role_mapping (
    role_id character varying(255) NOT NULL,
    user_id character varying(36) NOT NULL
);


ALTER TABLE public.user_role_mapping OWNER TO keycloak;

--
-- Name: web_origins; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.web_origins (
    client_id character varying(36) NOT NULL,
    value character varying(255) NOT NULL
);


ALTER TABLE public.web_origins OWNER TO keycloak;

--
-- Name: workflow_state; Type: TABLE; Schema: public; Owner: keycloak
--

CREATE TABLE public.workflow_state (
    execution_id character varying(255) NOT NULL,
    resource_id character varying(255) NOT NULL,
    workflow_id character varying(255) NOT NULL,
    resource_type character varying(255),
    scheduled_step_id character varying(255),
    scheduled_step_timestamp bigint
);


ALTER TABLE public.workflow_state OWNER TO keycloak;

--
-- Data for Name: admin_event_entity; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.admin_event_entity (id, admin_event_time, realm_id, operation_type, auth_realm_id, auth_client_id, auth_user_id, ip_address, resource_path, representation, error, resource_type, details_json) FROM stdin;
\.


--
-- Data for Name: associated_policy; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.associated_policy (policy_id, associated_policy_id) FROM stdin;
\.


--
-- Data for Name: authentication_execution; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.authentication_execution (id, alias, authenticator, realm_id, flow_id, requirement, priority, authenticator_flow, auth_flow_id, auth_config) FROM stdin;
c95637b3-386c-41fd-8af9-c4678e058540	\N	auth-cookie	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	e8e9d2ff-cf69-44b3-86f4-ea9cce5acea5	2	10	f	\N	\N
baba8438-f358-42ea-88d8-4c12923ee83f	\N	auth-spnego	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	e8e9d2ff-cf69-44b3-86f4-ea9cce5acea5	3	20	f	\N	\N
939d3506-e0a5-48e2-bab0-67a5f84c7bf6	\N	identity-provider-redirector	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	e8e9d2ff-cf69-44b3-86f4-ea9cce5acea5	2	25	f	\N	\N
15cc3fad-5c2f-45ec-849b-13e7f5f76c45	\N	\N	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	e8e9d2ff-cf69-44b3-86f4-ea9cce5acea5	2	30	t	2e72d788-2b2a-45db-9693-56925dd67dde	\N
71654bfb-3e83-4a08-89f6-be2f775e4929	\N	auth-username-password-form	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	2e72d788-2b2a-45db-9693-56925dd67dde	0	10	f	\N	\N
6eeacf59-b52f-464d-baee-0686a05d778b	\N	\N	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	2e72d788-2b2a-45db-9693-56925dd67dde	1	20	t	357fed5d-da44-4083-8119-f88ecf76fc59	\N
f1d23504-08a2-4440-bd79-489c4ee21ad5	\N	conditional-user-configured	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	357fed5d-da44-4083-8119-f88ecf76fc59	0	10	f	\N	\N
1e9f77a3-160a-4ee9-bb4e-9f553fdade83	\N	conditional-credential	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	357fed5d-da44-4083-8119-f88ecf76fc59	0	20	f	\N	3d88a67d-cd31-445d-aade-ffbb0799ade7
30092e6f-5f3d-40b2-a9a7-5987b8e50c24	\N	auth-otp-form	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	357fed5d-da44-4083-8119-f88ecf76fc59	2	30	f	\N	\N
df41a178-6ceb-4edb-ae33-37b675a0a2c6	\N	webauthn-authenticator	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	357fed5d-da44-4083-8119-f88ecf76fc59	3	40	f	\N	\N
ab2aa1be-37b6-4717-b1a8-1a259f94e555	\N	auth-recovery-authn-code-form	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	357fed5d-da44-4083-8119-f88ecf76fc59	3	50	f	\N	\N
24ae0a10-effa-47d2-88af-e1566c530421	\N	direct-grant-validate-username	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	abce7f50-bcfc-47fc-a115-a55caa56340c	0	10	f	\N	\N
4ad726b3-528f-4d31-bc48-5526314aacac	\N	direct-grant-validate-password	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	abce7f50-bcfc-47fc-a115-a55caa56340c	0	20	f	\N	\N
97b86bdb-f369-40ea-85b0-6fe032e74bf7	\N	\N	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	abce7f50-bcfc-47fc-a115-a55caa56340c	1	30	t	be7bc00a-376a-4489-aef9-10cf87d0e397	\N
74b965c3-1ef7-4d21-9896-c311f3816929	\N	conditional-user-configured	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	be7bc00a-376a-4489-aef9-10cf87d0e397	0	10	f	\N	\N
f294fff0-aa2a-4322-a68f-ce9bc8bf001d	\N	direct-grant-validate-otp	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	be7bc00a-376a-4489-aef9-10cf87d0e397	0	20	f	\N	\N
76979c4a-d563-4894-9b8a-5effd76a5199	\N	registration-page-form	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	c4f5bc43-fefb-4647-a4de-5c6e1a111d0c	0	10	t	21ef6559-5c26-4dee-aef4-144fd72039c3	\N
354ac2ec-d268-4284-ab2e-c1dbd2bc647a	\N	registration-user-creation	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	21ef6559-5c26-4dee-aef4-144fd72039c3	0	20	f	\N	\N
8bc57959-432c-49b8-a313-e6c7d023b1b2	\N	registration-password-action	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	21ef6559-5c26-4dee-aef4-144fd72039c3	0	50	f	\N	\N
c3a3ca28-e699-4613-b250-48a1c924a55a	\N	registration-recaptcha-action	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	21ef6559-5c26-4dee-aef4-144fd72039c3	3	60	f	\N	\N
fbe03707-8525-4973-be08-3bd2246f319f	\N	registration-terms-and-conditions	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	21ef6559-5c26-4dee-aef4-144fd72039c3	3	70	f	\N	\N
b243e0ee-924e-4324-83d9-3f71cca23ada	\N	reset-credentials-choose-user	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	cb636a91-3068-4711-95c7-981560c9f403	0	10	f	\N	\N
ea52dbd5-2d0f-44d8-b33a-e5e32d7bb666	\N	reset-credential-email	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	cb636a91-3068-4711-95c7-981560c9f403	0	20	f	\N	\N
db72a6d0-1f3c-4bfe-aaee-c3cf1c0300a5	\N	reset-password	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	cb636a91-3068-4711-95c7-981560c9f403	0	30	f	\N	\N
d5d2a895-690a-47ca-a5a6-945698520c19	\N	\N	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	cb636a91-3068-4711-95c7-981560c9f403	1	40	t	4c1b763d-956a-4d88-a5cb-74638e3543d2	\N
2862f6a9-2e8b-43e3-92d5-6c3bac1b234d	\N	conditional-user-configured	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	4c1b763d-956a-4d88-a5cb-74638e3543d2	0	10	f	\N	\N
bca764d3-e6d5-40bb-bc0d-2071b6a4a79c	\N	reset-otp	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	4c1b763d-956a-4d88-a5cb-74638e3543d2	0	20	f	\N	\N
4ffa8e4d-6a95-47f7-b798-239a8d80950f	\N	client-secret	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9fb38cd7-2140-4d5f-b1ef-06e83a5d1189	2	10	f	\N	\N
388c0bae-0199-47aa-bf34-a465a5e87104	\N	client-jwt	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9fb38cd7-2140-4d5f-b1ef-06e83a5d1189	2	20	f	\N	\N
4fc560d4-f38d-4695-b161-bf963ca6e12e	\N	client-secret-jwt	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9fb38cd7-2140-4d5f-b1ef-06e83a5d1189	2	30	f	\N	\N
ff63e7eb-a86a-404a-982e-60f72ba7a6e4	\N	client-x509	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9fb38cd7-2140-4d5f-b1ef-06e83a5d1189	2	40	f	\N	\N
30f411e0-59ca-4e55-b24c-65612bbfc6e7	\N	federated-jwt	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9fb38cd7-2140-4d5f-b1ef-06e83a5d1189	2	50	f	\N	\N
50ac58c8-cd9c-435f-a257-f6e609edfdb5	\N	idp-review-profile	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	2264a509-60f6-4146-894d-e4dc1cec61a3	0	10	f	\N	9e49b802-3870-4785-ad1b-8b563a55efcc
1df307c1-5df5-4e82-9cd3-645bea94f504	\N	\N	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	2264a509-60f6-4146-894d-e4dc1cec61a3	0	20	t	cec41da5-6d18-45a4-b302-d9f55d5d1fde	\N
b0744321-a6a5-41c3-a066-de096cdbb0b4	\N	idp-create-user-if-unique	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	cec41da5-6d18-45a4-b302-d9f55d5d1fde	2	10	f	\N	c8c02ac6-e0c3-41bc-a2af-dd936012020e
664db9d8-b3ad-4ea4-a3a6-d8a96e831fdd	\N	\N	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	cec41da5-6d18-45a4-b302-d9f55d5d1fde	2	20	t	c6a18a0e-5dc3-4e1a-b118-19991dc13a17	\N
74a64cd6-5bdb-48fd-a230-4bc98b2ffc89	\N	idp-confirm-link	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	c6a18a0e-5dc3-4e1a-b118-19991dc13a17	0	10	f	\N	\N
89d528ad-c639-4133-8a08-98b2488e2d0d	\N	\N	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	c6a18a0e-5dc3-4e1a-b118-19991dc13a17	0	20	t	514c9626-5acd-443d-85ad-888471032931	\N
13ea53d9-9dde-4f2d-a685-a5a5597678c2	\N	idp-email-verification	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	514c9626-5acd-443d-85ad-888471032931	2	10	f	\N	\N
5f91c5d6-e49a-42bb-8fa0-acbd6393e8ad	\N	\N	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	514c9626-5acd-443d-85ad-888471032931	2	20	t	f7511a0e-31d3-43f2-9cc5-95037952da7a	\N
20650d77-c0e7-461b-a6fa-c620b7963bfd	\N	idp-username-password-form	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	f7511a0e-31d3-43f2-9cc5-95037952da7a	0	10	f	\N	\N
545a8c6d-73ca-4f84-a574-f668fe6025b4	\N	\N	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	f7511a0e-31d3-43f2-9cc5-95037952da7a	1	20	t	3679963b-8dc1-4fd3-b617-bb7c66b2fb6b	\N
1ff1b474-d9ba-434f-b584-284d3808e012	\N	conditional-user-configured	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	3679963b-8dc1-4fd3-b617-bb7c66b2fb6b	0	10	f	\N	\N
e6378ef1-76f8-488f-9caa-e1046268f65c	\N	conditional-credential	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	3679963b-8dc1-4fd3-b617-bb7c66b2fb6b	0	20	f	\N	06149577-872b-44f5-b375-a04dbb4fd88c
b50b4cd8-99b4-40df-974a-28bfbc85e2f3	\N	auth-otp-form	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	3679963b-8dc1-4fd3-b617-bb7c66b2fb6b	2	30	f	\N	\N
cc2e9b35-4b2c-4512-a625-81993025ca30	\N	webauthn-authenticator	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	3679963b-8dc1-4fd3-b617-bb7c66b2fb6b	3	40	f	\N	\N
aabd1cae-6ff6-41c0-af7d-96544fa54e1d	\N	auth-recovery-authn-code-form	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	3679963b-8dc1-4fd3-b617-bb7c66b2fb6b	3	50	f	\N	\N
d0632ee6-ac9a-46e0-8598-e0500071dc8e	\N	http-basic-authenticator	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	d3837b44-18fe-4f86-89f3-d0e3650de352	0	10	f	\N	\N
4cd55fde-1342-41f2-8d92-0175286bf9ce	\N	docker-http-basic-authenticator	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	ccf84a7a-d7a7-473b-95cd-ca58ea113743	0	10	f	\N	\N
\.


--
-- Data for Name: authentication_flow; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.authentication_flow (id, alias, description, realm_id, provider_id, top_level, built_in) FROM stdin;
e8e9d2ff-cf69-44b3-86f4-ea9cce5acea5	browser	Browser based authentication	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	t	t
2e72d788-2b2a-45db-9693-56925dd67dde	forms	Username, password, otp and other auth forms.	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	f	t
357fed5d-da44-4083-8119-f88ecf76fc59	Browser - Conditional 2FA	Flow to determine if any 2FA is required for the authentication	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	f	t
abce7f50-bcfc-47fc-a115-a55caa56340c	direct grant	OpenID Connect Resource Owner Grant	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	t	t
be7bc00a-376a-4489-aef9-10cf87d0e397	Direct Grant - Conditional OTP	Flow to determine if the OTP is required for the authentication	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	f	t
c4f5bc43-fefb-4647-a4de-5c6e1a111d0c	registration	Registration flow	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	t	t
21ef6559-5c26-4dee-aef4-144fd72039c3	registration form	Registration form	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	form-flow	f	t
cb636a91-3068-4711-95c7-981560c9f403	reset credentials	Reset credentials for a user if they forgot their password or something	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	t	t
4c1b763d-956a-4d88-a5cb-74638e3543d2	Reset - Conditional OTP	Flow to determine if the OTP should be reset or not. Set to REQUIRED to force.	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	f	t
9fb38cd7-2140-4d5f-b1ef-06e83a5d1189	clients	Base authentication for clients	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	client-flow	t	t
2264a509-60f6-4146-894d-e4dc1cec61a3	first broker login	Actions taken after first broker login with identity provider account, which is not yet linked to any Keycloak account	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	t	t
cec41da5-6d18-45a4-b302-d9f55d5d1fde	User creation or linking	Flow for the existing/non-existing user alternatives	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	f	t
c6a18a0e-5dc3-4e1a-b118-19991dc13a17	Handle Existing Account	Handle what to do if there is existing account with same email/username like authenticated identity provider	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	f	t
514c9626-5acd-443d-85ad-888471032931	Account verification options	Method with which to verify the existing account	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	f	t
f7511a0e-31d3-43f2-9cc5-95037952da7a	Verify Existing Account by Re-authentication	Reauthentication of existing account	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	f	t
3679963b-8dc1-4fd3-b617-bb7c66b2fb6b	First broker login - Conditional 2FA	Flow to determine if any 2FA is required for the authentication	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	f	t
d3837b44-18fe-4f86-89f3-d0e3650de352	saml ecp	SAML ECP Profile Authentication Flow	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	t	t
ccf84a7a-d7a7-473b-95cd-ca58ea113743	docker auth	Used by Docker clients to authenticate against the IDP	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	basic-flow	t	t
\.


--
-- Data for Name: authenticator_config; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.authenticator_config (id, alias, realm_id) FROM stdin;
3d88a67d-cd31-445d-aade-ffbb0799ade7	browser-conditional-credential	eb86d28d-cdbb-4987-839a-7f8ed73f98ee
9e49b802-3870-4785-ad1b-8b563a55efcc	review profile config	eb86d28d-cdbb-4987-839a-7f8ed73f98ee
c8c02ac6-e0c3-41bc-a2af-dd936012020e	create unique user config	eb86d28d-cdbb-4987-839a-7f8ed73f98ee
06149577-872b-44f5-b375-a04dbb4fd88c	first-broker-login-conditional-credential	eb86d28d-cdbb-4987-839a-7f8ed73f98ee
\.


--
-- Data for Name: authenticator_config_entry; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.authenticator_config_entry (authenticator_id, value, name) FROM stdin;
06149577-872b-44f5-b375-a04dbb4fd88c	webauthn-passwordless	credentials
3d88a67d-cd31-445d-aade-ffbb0799ade7	webauthn-passwordless	credentials
9e49b802-3870-4785-ad1b-8b563a55efcc	missing	update.profile.on.first.login
c8c02ac6-e0c3-41bc-a2af-dd936012020e	false	require.password.update.after.registration
\.


--
-- Data for Name: broker_link; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.broker_link (identity_provider, storage_provider_id, realm_id, broker_user_id, broker_username, token, user_id) FROM stdin;
\.


--
-- Data for Name: client; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.client (id, enabled, full_scope_allowed, client_id, not_before, public_client, secret, base_url, bearer_only, management_url, surrogate_auth_required, realm_id, protocol, node_rereg_timeout, frontchannel_logout, consent_required, name, service_accounts_enabled, client_authenticator_type, root_url, description, registration_token, standard_flow_enabled, implicit_flow_enabled, direct_access_grants_enabled, always_display_in_console) FROM stdin;
9e64fa07-4221-4426-b056-6d4e83cc7021	t	f	master-realm	0	f	\N	\N	t	\N	f	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	\N	0	f	f	master Realm	f	client-secret	\N	\N	\N	t	f	f	f
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	t	f	account	0	t	\N	/realms/master/account/	f	\N	f	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	openid-connect	0	f	f	${client_account}	f	client-secret	${authBaseUrl}	\N	\N	t	f	f	f
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	t	f	account-console	0	t	\N	/realms/master/account/	f	\N	f	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	openid-connect	0	f	f	${client_account-console}	f	client-secret	${authBaseUrl}	\N	\N	t	f	f	f
46baaef2-287a-4ef6-bbe2-39491e649624	t	f	broker	0	f	\N	\N	t	\N	f	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	openid-connect	0	f	f	${client_broker}	f	client-secret	\N	\N	\N	t	f	f	f
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	t	t	security-admin-console	0	t	\N	/admin/master/console/	f	\N	f	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	openid-connect	0	f	f	${client_security-admin-console}	f	client-secret	${authAdminUrl}	\N	\N	t	f	f	f
17b26dc8-4092-47c6-bb03-af98e7f43514	t	t	admin-cli	0	t	\N	\N	f	\N	f	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	openid-connect	0	f	f	${client_admin-cli}	f	client-secret	\N	\N	\N	f	f	t	f
9be1e3f2-527b-4899-b53c-123a908c9d0d	t	t	backrest	0	f	3jpW4nLOccDiEPm1t1XnbCNH2GKNTsuz		f		f	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	openid-connect	-1	t	f	backrest	f	client-secret			\N	t	f	f	f
\.


--
-- Data for Name: client_attributes; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.client_attributes (client_id, name, value) FROM stdin;
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	post.logout.redirect.uris	+
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	post.logout.redirect.uris	+
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	pkce.code.challenge.method	S256
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	post.logout.redirect.uris	+
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	pkce.code.challenge.method	S256
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	client.use.lightweight.access.token.enabled	true
17b26dc8-4092-47c6-bb03-af98e7f43514	client.use.lightweight.access.token.enabled	true
9be1e3f2-527b-4899-b53c-123a908c9d0d	standard.token.exchange.enabled	false
9be1e3f2-527b-4899-b53c-123a908c9d0d	oauth2.jwt.authorization.grant.enabled	false
9be1e3f2-527b-4899-b53c-123a908c9d0d	oauth2.device.authorization.grant.enabled	false
9be1e3f2-527b-4899-b53c-123a908c9d0d	oidc.ciba.grant.enabled	false
9be1e3f2-527b-4899-b53c-123a908c9d0d	dpop.bound.access.tokens	false
9be1e3f2-527b-4899-b53c-123a908c9d0d	backchannel.logout.session.required	true
9be1e3f2-527b-4899-b53c-123a908c9d0d	backchannel.logout.revoke.offline.tokens	false
9be1e3f2-527b-4899-b53c-123a908c9d0d	client.secret.creation.time	1781917289
\.


--
-- Data for Name: client_auth_flow_bindings; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.client_auth_flow_bindings (client_id, flow_id, binding_name) FROM stdin;
\.


--
-- Data for Name: client_initial_access; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.client_initial_access (id, realm_id, "timestamp", expiration, count, remaining_count) FROM stdin;
\.


--
-- Data for Name: client_node_registrations; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.client_node_registrations (client_id, value, name) FROM stdin;
\.


--
-- Data for Name: client_scope; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.client_scope (id, name, realm_id, description, protocol) FROM stdin;
b511e099-6962-4dda-bdef-b9f25ce57db8	offline_access	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	OpenID Connect built-in scope: offline_access	openid-connect
48d8e6fc-1e74-4bcc-bd96-8e90f7ad329d	role_list	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	SAML role list	saml
251977c1-46b1-4844-bb5a-425e2212ef0e	saml_organization	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	Organization Membership	saml
3a505778-d76c-4957-af68-9019361f5fc9	profile	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	OpenID Connect built-in scope: profile	openid-connect
a9d7c3c0-0351-4d08-aa53-eeed1f01e858	email	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	OpenID Connect built-in scope: email	openid-connect
6b85c7f3-6922-4d08-94a1-38276d9cc804	address	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	OpenID Connect built-in scope: address	openid-connect
8fc9262c-e47a-47e5-879b-4e1df207ea2f	phone	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	OpenID Connect built-in scope: phone	openid-connect
dff93946-cd22-41b2-867d-350628a3e044	roles	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	OpenID Connect scope for add user roles to the access token	openid-connect
16b30cc6-85ff-4ba9-bb1f-85f802271826	web-origins	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	OpenID Connect scope for add allowed web origins to the access token	openid-connect
6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd	microprofile-jwt	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	Microprofile - JWT built-in scope	openid-connect
3ae652cf-ae49-4be9-9c9d-70ceef556823	acr	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	OpenID Connect scope for add acr (authentication context class reference) to the token	openid-connect
eff33a74-12c5-460d-94dc-5ef255d0a1c2	basic	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	OpenID Connect scope for add all basic claims to the token	openid-connect
37d2fdba-cb42-41ca-b47e-3439a724dcdf	service_account	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	Specific scope for a client enabled for service accounts	openid-connect
a7467a26-8787-424e-8615-765968eb03a2	organization	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	Additional claims about the organization a subject belongs to	openid-connect
\.


--
-- Data for Name: client_scope_attributes; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.client_scope_attributes (scope_id, value, name) FROM stdin;
b511e099-6962-4dda-bdef-b9f25ce57db8	true	display.on.consent.screen
b511e099-6962-4dda-bdef-b9f25ce57db8	${offlineAccessScopeConsentText}	consent.screen.text
48d8e6fc-1e74-4bcc-bd96-8e90f7ad329d	true	display.on.consent.screen
48d8e6fc-1e74-4bcc-bd96-8e90f7ad329d	${samlRoleListScopeConsentText}	consent.screen.text
251977c1-46b1-4844-bb5a-425e2212ef0e	false	display.on.consent.screen
3a505778-d76c-4957-af68-9019361f5fc9	true	display.on.consent.screen
3a505778-d76c-4957-af68-9019361f5fc9	${profileScopeConsentText}	consent.screen.text
3a505778-d76c-4957-af68-9019361f5fc9	true	include.in.token.scope
a9d7c3c0-0351-4d08-aa53-eeed1f01e858	true	display.on.consent.screen
a9d7c3c0-0351-4d08-aa53-eeed1f01e858	${emailScopeConsentText}	consent.screen.text
a9d7c3c0-0351-4d08-aa53-eeed1f01e858	true	include.in.token.scope
6b85c7f3-6922-4d08-94a1-38276d9cc804	true	display.on.consent.screen
6b85c7f3-6922-4d08-94a1-38276d9cc804	${addressScopeConsentText}	consent.screen.text
6b85c7f3-6922-4d08-94a1-38276d9cc804	true	include.in.token.scope
8fc9262c-e47a-47e5-879b-4e1df207ea2f	true	display.on.consent.screen
8fc9262c-e47a-47e5-879b-4e1df207ea2f	${phoneScopeConsentText}	consent.screen.text
8fc9262c-e47a-47e5-879b-4e1df207ea2f	true	include.in.token.scope
dff93946-cd22-41b2-867d-350628a3e044	true	display.on.consent.screen
dff93946-cd22-41b2-867d-350628a3e044	${rolesScopeConsentText}	consent.screen.text
dff93946-cd22-41b2-867d-350628a3e044	false	include.in.token.scope
16b30cc6-85ff-4ba9-bb1f-85f802271826	false	display.on.consent.screen
16b30cc6-85ff-4ba9-bb1f-85f802271826		consent.screen.text
16b30cc6-85ff-4ba9-bb1f-85f802271826	false	include.in.token.scope
6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd	false	display.on.consent.screen
6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd	true	include.in.token.scope
3ae652cf-ae49-4be9-9c9d-70ceef556823	false	display.on.consent.screen
3ae652cf-ae49-4be9-9c9d-70ceef556823	false	include.in.token.scope
eff33a74-12c5-460d-94dc-5ef255d0a1c2	false	display.on.consent.screen
eff33a74-12c5-460d-94dc-5ef255d0a1c2	false	include.in.token.scope
37d2fdba-cb42-41ca-b47e-3439a724dcdf	false	display.on.consent.screen
37d2fdba-cb42-41ca-b47e-3439a724dcdf	false	include.in.token.scope
a7467a26-8787-424e-8615-765968eb03a2	true	display.on.consent.screen
a7467a26-8787-424e-8615-765968eb03a2	${organizationScopeConsentText}	consent.screen.text
a7467a26-8787-424e-8615-765968eb03a2	true	include.in.token.scope
\.


--
-- Data for Name: client_scope_client; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.client_scope_client (client_id, scope_id, default_scope) FROM stdin;
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	dff93946-cd22-41b2-867d-350628a3e044	t
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	a9d7c3c0-0351-4d08-aa53-eeed1f01e858	t
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	3ae652cf-ae49-4be9-9c9d-70ceef556823	t
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	eff33a74-12c5-460d-94dc-5ef255d0a1c2	t
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	16b30cc6-85ff-4ba9-bb1f-85f802271826	t
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	3a505778-d76c-4957-af68-9019361f5fc9	t
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd	f
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	8fc9262c-e47a-47e5-879b-4e1df207ea2f	f
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	a7467a26-8787-424e-8615-765968eb03a2	f
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	6b85c7f3-6922-4d08-94a1-38276d9cc804	f
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	b511e099-6962-4dda-bdef-b9f25ce57db8	f
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	dff93946-cd22-41b2-867d-350628a3e044	t
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	a9d7c3c0-0351-4d08-aa53-eeed1f01e858	t
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	3ae652cf-ae49-4be9-9c9d-70ceef556823	t
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	eff33a74-12c5-460d-94dc-5ef255d0a1c2	t
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	16b30cc6-85ff-4ba9-bb1f-85f802271826	t
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	3a505778-d76c-4957-af68-9019361f5fc9	t
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd	f
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	8fc9262c-e47a-47e5-879b-4e1df207ea2f	f
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	a7467a26-8787-424e-8615-765968eb03a2	f
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	6b85c7f3-6922-4d08-94a1-38276d9cc804	f
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	b511e099-6962-4dda-bdef-b9f25ce57db8	f
17b26dc8-4092-47c6-bb03-af98e7f43514	dff93946-cd22-41b2-867d-350628a3e044	t
17b26dc8-4092-47c6-bb03-af98e7f43514	a9d7c3c0-0351-4d08-aa53-eeed1f01e858	t
17b26dc8-4092-47c6-bb03-af98e7f43514	3ae652cf-ae49-4be9-9c9d-70ceef556823	t
17b26dc8-4092-47c6-bb03-af98e7f43514	eff33a74-12c5-460d-94dc-5ef255d0a1c2	t
17b26dc8-4092-47c6-bb03-af98e7f43514	16b30cc6-85ff-4ba9-bb1f-85f802271826	t
17b26dc8-4092-47c6-bb03-af98e7f43514	3a505778-d76c-4957-af68-9019361f5fc9	t
17b26dc8-4092-47c6-bb03-af98e7f43514	6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd	f
17b26dc8-4092-47c6-bb03-af98e7f43514	8fc9262c-e47a-47e5-879b-4e1df207ea2f	f
17b26dc8-4092-47c6-bb03-af98e7f43514	a7467a26-8787-424e-8615-765968eb03a2	f
17b26dc8-4092-47c6-bb03-af98e7f43514	6b85c7f3-6922-4d08-94a1-38276d9cc804	f
17b26dc8-4092-47c6-bb03-af98e7f43514	b511e099-6962-4dda-bdef-b9f25ce57db8	f
46baaef2-287a-4ef6-bbe2-39491e649624	dff93946-cd22-41b2-867d-350628a3e044	t
46baaef2-287a-4ef6-bbe2-39491e649624	a9d7c3c0-0351-4d08-aa53-eeed1f01e858	t
46baaef2-287a-4ef6-bbe2-39491e649624	3ae652cf-ae49-4be9-9c9d-70ceef556823	t
46baaef2-287a-4ef6-bbe2-39491e649624	eff33a74-12c5-460d-94dc-5ef255d0a1c2	t
46baaef2-287a-4ef6-bbe2-39491e649624	16b30cc6-85ff-4ba9-bb1f-85f802271826	t
46baaef2-287a-4ef6-bbe2-39491e649624	3a505778-d76c-4957-af68-9019361f5fc9	t
46baaef2-287a-4ef6-bbe2-39491e649624	6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd	f
46baaef2-287a-4ef6-bbe2-39491e649624	8fc9262c-e47a-47e5-879b-4e1df207ea2f	f
46baaef2-287a-4ef6-bbe2-39491e649624	a7467a26-8787-424e-8615-765968eb03a2	f
46baaef2-287a-4ef6-bbe2-39491e649624	6b85c7f3-6922-4d08-94a1-38276d9cc804	f
46baaef2-287a-4ef6-bbe2-39491e649624	b511e099-6962-4dda-bdef-b9f25ce57db8	f
9e64fa07-4221-4426-b056-6d4e83cc7021	dff93946-cd22-41b2-867d-350628a3e044	t
9e64fa07-4221-4426-b056-6d4e83cc7021	a9d7c3c0-0351-4d08-aa53-eeed1f01e858	t
9e64fa07-4221-4426-b056-6d4e83cc7021	3ae652cf-ae49-4be9-9c9d-70ceef556823	t
9e64fa07-4221-4426-b056-6d4e83cc7021	eff33a74-12c5-460d-94dc-5ef255d0a1c2	t
9e64fa07-4221-4426-b056-6d4e83cc7021	16b30cc6-85ff-4ba9-bb1f-85f802271826	t
9e64fa07-4221-4426-b056-6d4e83cc7021	3a505778-d76c-4957-af68-9019361f5fc9	t
9e64fa07-4221-4426-b056-6d4e83cc7021	6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd	f
9e64fa07-4221-4426-b056-6d4e83cc7021	8fc9262c-e47a-47e5-879b-4e1df207ea2f	f
9e64fa07-4221-4426-b056-6d4e83cc7021	a7467a26-8787-424e-8615-765968eb03a2	f
9e64fa07-4221-4426-b056-6d4e83cc7021	6b85c7f3-6922-4d08-94a1-38276d9cc804	f
9e64fa07-4221-4426-b056-6d4e83cc7021	b511e099-6962-4dda-bdef-b9f25ce57db8	f
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	dff93946-cd22-41b2-867d-350628a3e044	t
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	a9d7c3c0-0351-4d08-aa53-eeed1f01e858	t
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	3ae652cf-ae49-4be9-9c9d-70ceef556823	t
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	eff33a74-12c5-460d-94dc-5ef255d0a1c2	t
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	16b30cc6-85ff-4ba9-bb1f-85f802271826	t
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	3a505778-d76c-4957-af68-9019361f5fc9	t
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd	f
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	8fc9262c-e47a-47e5-879b-4e1df207ea2f	f
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	a7467a26-8787-424e-8615-765968eb03a2	f
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	6b85c7f3-6922-4d08-94a1-38276d9cc804	f
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	b511e099-6962-4dda-bdef-b9f25ce57db8	f
9be1e3f2-527b-4899-b53c-123a908c9d0d	dff93946-cd22-41b2-867d-350628a3e044	t
9be1e3f2-527b-4899-b53c-123a908c9d0d	a9d7c3c0-0351-4d08-aa53-eeed1f01e858	t
9be1e3f2-527b-4899-b53c-123a908c9d0d	3ae652cf-ae49-4be9-9c9d-70ceef556823	t
9be1e3f2-527b-4899-b53c-123a908c9d0d	eff33a74-12c5-460d-94dc-5ef255d0a1c2	t
9be1e3f2-527b-4899-b53c-123a908c9d0d	16b30cc6-85ff-4ba9-bb1f-85f802271826	t
9be1e3f2-527b-4899-b53c-123a908c9d0d	3a505778-d76c-4957-af68-9019361f5fc9	t
9be1e3f2-527b-4899-b53c-123a908c9d0d	6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd	f
9be1e3f2-527b-4899-b53c-123a908c9d0d	8fc9262c-e47a-47e5-879b-4e1df207ea2f	f
9be1e3f2-527b-4899-b53c-123a908c9d0d	a7467a26-8787-424e-8615-765968eb03a2	f
9be1e3f2-527b-4899-b53c-123a908c9d0d	6b85c7f3-6922-4d08-94a1-38276d9cc804	f
9be1e3f2-527b-4899-b53c-123a908c9d0d	b511e099-6962-4dda-bdef-b9f25ce57db8	f
\.


--
-- Data for Name: client_scope_role_mapping; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.client_scope_role_mapping (scope_id, role_id) FROM stdin;
b511e099-6962-4dda-bdef-b9f25ce57db8	10fbb334-c415-430e-9b09-496038b00760
\.


--
-- Data for Name: component; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.component (id, name, parent_id, provider_id, provider_type, realm_id, sub_type) FROM stdin;
d1689f5a-4727-4061-a85f-e3c0425f65c5	Trusted Hosts	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	trusted-hosts	org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	anonymous
1035f8e7-ce69-44e6-b86f-2cb38da61b4e	Consent Required	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	consent-required	org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	anonymous
cec7ac28-7f9e-4cc6-a51e-1f535dcbd9e7	Full Scope Disabled	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	scope	org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	anonymous
e07e4cde-e727-4138-b0b1-ee2b180de7d4	Max Clients Limit	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	max-clients	org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	anonymous
5c8bbf78-548d-4fa8-b16a-58e283ec4b53	Allowed Protocol Mapper Types	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	allowed-protocol-mappers	org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	anonymous
8a0bc03e-f2cc-411f-ba6b-ab7bf9b6867f	Allowed Client Scopes	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	allowed-client-templates	org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	anonymous
3cffda35-204d-46a1-9a05-357cdfef040d	Allowed Registration Web Origins	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	registration-web-origins	org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	anonymous
e5fd421d-d304-4eeb-a9eb-3033860b0db2	Allowed Protocol Mapper Types	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	allowed-protocol-mappers	org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	authenticated
13c118c4-4de9-4258-9a44-e8884aabf341	Allowed Client Scopes	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	allowed-client-templates	org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	authenticated
805e5fad-0e62-409b-b832-c95150603d31	Allowed Registration Web Origins	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	registration-web-origins	org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	authenticated
81ee0c62-d2e8-4b83-b02d-da979dae9d94	rsa-generated	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	rsa-generated	org.keycloak.keys.KeyProvider	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	\N
1e5cc8cc-9dd3-4e54-912b-af3a04910478	rsa-enc-generated	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	rsa-enc-generated	org.keycloak.keys.KeyProvider	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	\N
4736edc8-96b2-4f9c-964d-9e7dab814e6a	hmac-generated-hs512	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	hmac-generated	org.keycloak.keys.KeyProvider	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	\N
a8238823-5fd6-42d6-828a-f6677e91e018	aes-generated	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	aes-generated	org.keycloak.keys.KeyProvider	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	\N
25d469ec-870a-46da-af50-4a67fa85dc54	\N	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	declarative-user-profile	org.keycloak.userprofile.UserProfileProvider	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	\N
\.


--
-- Data for Name: component_config; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.component_config (id, component_id, name, value) FROM stdin;
41174b9a-3d9c-45de-aedf-fdf2e87c46e5	e07e4cde-e727-4138-b0b1-ee2b180de7d4	max-clients	200
3c7c0f26-a7c9-4f13-ab2b-15b1b35b1b2d	13c118c4-4de9-4258-9a44-e8884aabf341	allow-default-scopes	true
efc2aebb-e484-4de0-a120-aa11a04c5001	e5fd421d-d304-4eeb-a9eb-3033860b0db2	allowed-protocol-mapper-types	saml-user-attribute-mapper
18adbe22-f8fc-487f-ae80-d8f3acaaa907	e5fd421d-d304-4eeb-a9eb-3033860b0db2	allowed-protocol-mapper-types	oidc-usermodel-property-mapper
1de97fb3-cc85-4ce9-af52-10e9aff2bbd3	e5fd421d-d304-4eeb-a9eb-3033860b0db2	allowed-protocol-mapper-types	oidc-address-mapper
5d5b78af-9bfc-4ed3-a983-6484a2a7b466	e5fd421d-d304-4eeb-a9eb-3033860b0db2	allowed-protocol-mapper-types	oidc-sha256-pairwise-sub-mapper
406fc971-59c4-4773-8a4d-29374de54909	e5fd421d-d304-4eeb-a9eb-3033860b0db2	allowed-protocol-mapper-types	saml-user-property-mapper
c6fc4d7a-c028-43f0-be7c-ffc8421dcf2f	e5fd421d-d304-4eeb-a9eb-3033860b0db2	allowed-protocol-mapper-types	saml-role-list-mapper
6e43d3b9-c7ac-45f0-8629-3ebe6064add6	e5fd421d-d304-4eeb-a9eb-3033860b0db2	allowed-protocol-mapper-types	oidc-full-name-mapper
2a51616d-312a-4ab2-a0e3-30d842768f40	e5fd421d-d304-4eeb-a9eb-3033860b0db2	allowed-protocol-mapper-types	oidc-usermodel-attribute-mapper
3a7fb211-a339-4daf-9007-764a406a9127	d1689f5a-4727-4061-a85f-e3c0425f65c5	client-uris-must-match	true
0e2be6de-d9cc-423e-b17c-98dada1e29b2	d1689f5a-4727-4061-a85f-e3c0425f65c5	host-sending-registration-request-must-match	true
63695e55-a85b-4e7c-8895-7ce2b937631b	8a0bc03e-f2cc-411f-ba6b-ab7bf9b6867f	allow-default-scopes	true
6b19d0b9-9168-4bc3-aa3a-df66411754c0	5c8bbf78-548d-4fa8-b16a-58e283ec4b53	allowed-protocol-mapper-types	oidc-sha256-pairwise-sub-mapper
b5dd1fb6-9548-415c-923e-3bac12a1f6ae	5c8bbf78-548d-4fa8-b16a-58e283ec4b53	allowed-protocol-mapper-types	oidc-usermodel-property-mapper
90b6dc6a-2fa2-4d1a-8809-52aec5f352ef	5c8bbf78-548d-4fa8-b16a-58e283ec4b53	allowed-protocol-mapper-types	saml-role-list-mapper
739ad6d6-5fcf-40a5-90e6-e913570cdd0b	5c8bbf78-548d-4fa8-b16a-58e283ec4b53	allowed-protocol-mapper-types	saml-user-attribute-mapper
ca37f1f3-0308-4fab-8928-7fa5803dbd89	5c8bbf78-548d-4fa8-b16a-58e283ec4b53	allowed-protocol-mapper-types	oidc-address-mapper
627613c3-72e2-47de-a80e-7ef5c77d0722	5c8bbf78-548d-4fa8-b16a-58e283ec4b53	allowed-protocol-mapper-types	oidc-full-name-mapper
81a14bd4-c519-438c-95ad-e2929752163c	5c8bbf78-548d-4fa8-b16a-58e283ec4b53	allowed-protocol-mapper-types	oidc-usermodel-attribute-mapper
53ccf54f-59ea-4d34-95d4-1b6cddf2f4bb	5c8bbf78-548d-4fa8-b16a-58e283ec4b53	allowed-protocol-mapper-types	saml-user-property-mapper
49c5c02d-2ffb-4fb6-bbea-4855ed89beb4	25d469ec-870a-46da-af50-4a67fa85dc54	kc.user.profile.config	{"attributes":[{"name":"username","displayName":"${username}","validations":{"length":{"min":3,"max":255},"username-prohibited-characters":{},"up-username-not-idn-homograph":{}},"permissions":{"view":["admin","user"],"edit":["admin","user"]},"multivalued":false},{"name":"email","displayName":"${email}","validations":{"email":{},"length":{"max":255}},"permissions":{"view":["admin","user"],"edit":["admin","user"]},"multivalued":false},{"name":"firstName","displayName":"${firstName}","validations":{"length":{"max":255},"person-name-prohibited-characters":{}},"permissions":{"view":["admin","user"],"edit":["admin","user"]},"multivalued":false},{"name":"lastName","displayName":"${lastName}","validations":{"length":{"max":255},"person-name-prohibited-characters":{}},"permissions":{"view":["admin","user"],"edit":["admin","user"]},"multivalued":false}],"groups":[{"name":"user-metadata","displayHeader":"User metadata","displayDescription":"Attributes, which refer to user metadata"}]}
8848faad-71f2-46d5-a453-ceb2cb14f7a5	1e5cc8cc-9dd3-4e54-912b-af3a04910478	algorithm	RSA-OAEP
31413fb9-cadb-4a23-86bb-c56583537569	1e5cc8cc-9dd3-4e54-912b-af3a04910478	keyUse	ENC
d298abd4-e108-4410-8ffc-9f22c3b2167b	1e5cc8cc-9dd3-4e54-912b-af3a04910478	privateKey	MIIEogIBAAKCAQEApVDUNHKPX9N6E+Fwyvce/rjwwNi78Up/1kNycx/8GLN/TeTMuIvWowO8XCf90AsKDCL9tu1/2FkWWvCq6496Pvql9boKdkBM+hPc8ioMzxZwzhXxVbgDVN//kOFJGelodABH8ep1FvnrSM1iQGMLwB9g0z+zoU9tA2smFLZd2imhd9dKk2gdhbasRjmXfk3B4apLwUlICp4A2K/lsC7+NdxS5qhmJU6J79r7UIatc6Kqmtfv9j4sP1AIB8unaKk51wQ8BOwm8y73Mzw0Q7JHXCXnIV9xOJZ/PnQ9MD30orAt9pkP5qbn+Zp/EGqnLhYHRqRCyAat9Z5IfTjglbF8ewIDAQABAoIBAAraCXP/6SVzLlpLvCm2mxRBc5xVHdEzAL1B5CtmeBfvAHZOhJnApDBDOIQcI+8aKmiti1YMtQ2wm2UQ00dvPakQrwA4XCNzCRqJX0GOPRUC9hixHAxybdWOdqo9/5xx0+d5dT+OEm2VrjozMTXkyoqsBKEZV2NJYXCOAgvuBK8jXW8ks/Mc8e9WjClBQnCQWaOcZxjQdXdRXPmu8Thgf7yR4+zCeZ2C/EBvMZ5jj9T7ykgi2Tj73FrgcZWo1Q3uPUlHTpC8zCn74NreXZMWMHe3H+WfV81+/nsJZgdx54/1lEoz+hDHsbH8b3CA21XBAYotMnVVA5zFdSzbz9gTp5ECgYEA06QZJg5tlwE53FBJEK65y+xK0+rAqn2mGfAp65ECKurVSOY8+FVqNhlvWDiK/e1+KkNnZZ8eUiOQkZvf4y9kNv95W62KhJdqPdmdM58qGNoRb6sf5QrhcNYU2nU0m8eHSImbPVLkTIIvbMTefb3/boog2iiXJG+QvjIy83OnG/8CgYEAx/cU9Ugb5/k2RD/hyXgspOl4D92zcaXLBB+hxcvDQLkE3gvws+mQMsk+K6DHi0+oDlSEGM+ApmmJKiCPXmvGEb9Y5koW9WJPzUzQXfuLqNxRL+PJZoUiq4FA+TvQZfSuDvmIGfDYAGjkBjgC8qel0CBxCu1NcWBhctQRqrrED4UCgYAxlvW9kQvkogjosncsTYSDX6540Tyrth1BXqCz7ZpQbA3lsuz+UyU73+HTDgyjSw6Q4JJNoWb9YA/zzk47cVNN/7Zz4MngH4ppS6AmBFlc0VvcioBCrrX8Nm1UcroM9kegm32gdNfBhY+PMOHhHK/JOtxPwcsIYovLxP9jQ+oYCQKBgCaEobTdvwJgLuPWqld0nqTllAr6WaZ6mTCiJzdCmMnO2fEru6HsS6p8uU0OG2HqZiTcgtWEovNrQNCslPFLMUwZ37X8b4+08EpLkZeI7M4KIllnN1RYMOV4cFuR+gKprx2TU3QrwG/TxJiuEWNMh9Qfa6b/Lvu35Q2JbB3G/B1BAoGADKQb5Urrogad02WFzneOUGdvyyCiNqDujodGUmzu7ivb35T6/zNdK/ReQcuZal2z7URCT79V0d/Ecvt6i55co5+k41JrBylNxblGlctAVQMkgyDnJMsg4d1J0OG93wS5KmexhNWAbzYNZ9A8ybpiv9AtNz5JToSrP+C63cSKQZI=
9449540f-a6e8-4167-b44b-f637e39aa9c4	1e5cc8cc-9dd3-4e54-912b-af3a04910478	certificate	MIICmzCCAYMCBgGe4oZZGDANBgkqhkiG9w0BAQsFADARMQ8wDQYDVQQDDAZtYXN0ZXIwHhcNMjYwNjIwMDA1MzM1WhcNMzYwNjIwMDA1NTE1WjARMQ8wDQYDVQQDDAZtYXN0ZXIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQClUNQ0co9f03oT4XDK9x7+uPDA2LvxSn/WQ3JzH/wYs39N5My4i9ajA7xcJ/3QCwoMIv227X/YWRZa8Krrj3o++qX1ugp2QEz6E9zyKgzPFnDOFfFVuANU3/+Q4UkZ6Wh0AEfx6nUW+etIzWJAYwvAH2DTP7OhT20DayYUtl3aKaF310qTaB2FtqxGOZd+TcHhqkvBSUgKngDYr+WwLv413FLmqGYlTonv2vtQhq1zoqqa1+/2Piw/UAgHy6doqTnXBDwE7CbzLvczPDRDskdcJechX3E4ln8+dD0wPfSisC32mQ/mpuf5mn8QaqcuFgdGpELIBq31nkh9OOCVsXx7AgMBAAEwDQYJKoZIhvcNAQELBQADggEBACQZ9G47f/IOkMhNLh9rRq1L2bxTz/jfUVyX5R+Ud98wrLVFenOZ03fzWoOP+DAm0PWaw4Q+971+Pq3QBAePU47NLGpPQ/jXaheOVjEdYB5nj/Z+R7sM/RJK4AVZ+o/b0mXtKtsNhDfZMSw9kOREQAEiMMkNmBHdJryF2TukqshctcQvA3DgKptqRtg6tPGGaZUsiDx8SSk7nqmb/swCS8mUGdhKkLTMmLK7i/PapcZH9CgKPNXDwERErJo6jTjaFhnyCIIC6CqGA00weuUvY94zlDilkQ5FU1+KPiMkQruZJ5kxUlXr9SwWQbjrNUylcGouDCORuvce4tKIZsIq4Pk=
f2e301ac-f791-4ab4-b82b-8c7f0f877082	1e5cc8cc-9dd3-4e54-912b-af3a04910478	priority	100
d135d232-4afa-4295-ba5d-9e3b774763b1	4736edc8-96b2-4f9c-964d-9e7dab814e6a	secret	K-lKg8xZX1nJasKjjmabHaWvpbi2eR6K49eZkOXEyPCf9DaFlPg8rvQM1sCP3HRGyKHRdmn6vgLqJoGcwxQB6rcm8clcYVe0JBGR_dvzT7aW3sb3_S-Cy6FCtts7PIEHGIhsnxHWqv4LWFMHcUbsKWde8Ns0XjE9eDX848Gvf7M
4cc56485-3d60-4c36-a6c3-8137c4d3a99d	4736edc8-96b2-4f9c-964d-9e7dab814e6a	kid	3112aed9-51b0-4e7d-b4c7-de516eba13cc
e66e7a33-cb8e-42d5-b6d2-e27dfb3c8de6	4736edc8-96b2-4f9c-964d-9e7dab814e6a	algorithm	HS512
ad5db697-18c2-4dbc-8b5e-143aeaa469b7	4736edc8-96b2-4f9c-964d-9e7dab814e6a	priority	100
84afbfa0-bc48-46f8-9eea-c3f20a27baea	81ee0c62-d2e8-4b83-b02d-da979dae9d94	privateKey	MIIEpAIBAAKCAQEA24mCIv9/IM87JFWP10zQTOA+2mWWOX/z80sgQ8yqggJGe6CpJLHcMGjQLkRmeRJ2diX+4+hQifaKaJmJAvsK00lX1rZUcaiRgW9J24GOqdB3TVFFVqZbBgdPacIShmsrHY1q2MRNamsAIrp6iNSkpdKbwSfw97Dk3xu6JwWHMwnvrjxwfU9ENPDLROhQj+oTd/49pIYKa/uCXk0evO/h+LkFhWZQvAkOnzEKx4+azhPQU4nnkXTkBcQthXsmh8zV7aRUYOwMh0Y+37q8lCmn8TpuZYY0MBnFn5A3lP6Z8E1YJ0OEERG4Gzd46g3C+ZnETM6qnYxwEuQyvF2X8tZTkQIDAQABAoIBABuYGZzni2Wm1pK3FHjl5Uq8ZvoRicPZcuLSPxB2kbn8qjpQ0+HSX1BQZFZkb5LpQK2SEgs4gKMOs4/5OHEA9/fdwKYyzcNpgjyILfriunlaxBwaSoJdL5S+53ruE2EE6GrzAwqTBf4Jy/8RfGRlgfp70esB57ZibCok9I2CEt/VtMO2eu0VzxzozI/wgzBdeSzShcYfemxh5776g63rHN5mdcCE2QABC0FVf4iA/XnmWS6GmduCQ19YvYJNDWe54Zw9+5dyNEhZ3g0xukDcncNHGitYVobKWVG439WeZQV1dX/0o1sFLsl4rJB57oFwe8xuiIFBqbYxXo7ptcPqQL8CgYEA9MWYjAkbgY7yqq39UDFNTqLvHUir1QyVXAKgvOu1YOjBZdsyEEZNOV6thcOWNKgGxPb85eVqcD8rtP3KwDOgTNhSheB6fECpf2ep1Tqy5CZhQj52q4pPBIph0ghdGTR+nzF7SAw+9518BnkxYkDTGn2MDJpl4q3OdpiC34wEfpMCgYEA5ZuTgqLDrG/T2O7lT28r7PWjOQOhoXLX36Ere9THHtOK6QYQ1qlaFikypRJgu2G9k7Oy1NLuMOjK8erDCiz5+kdh4Pq9Tr9X4io9VJlo4of9US7UbbcPMuLMfFBRbXhexHoXAYa5vZidiLAGmNYFKwgzALh7YQlPk+LVBxH2V8sCgYB9LYByPUYf3+ciepCNrmkGyjTXGQ8niaPoxj9F3pWH1gDyAkN8JzffGxhKzFfI3hV4LYfwWn8woF3N1e0WllBofEjXxpFdcgQ879re/YH3Q9mBc0hlOfpnLA4Sx8w006/d4gRWOE0LfTKsbNZglR5g8cvpthxc7N4lsKFdidHSmQKBgQCb2VHy5g/zR6SAJwe3NzqViNXVqUVYlN86h+dAElll7yjmqptbWXAwgp5BtYu7JMjNLLhCBTpwMFMwia0Bhy3WWAVz9D3y6aX5ebpPZiHvQWcMZ8EBB7RlUdSCvPHIYF/S9RwQiSYiLyke0nxn2T2Ay/vyjUYRw4QkWV9HgUW4qQKBgQCHyvwfkba7aS7yE9zL1XpLyWwFO2NcRVYpS2j6n/DO+zioeEPbwkibmEESTLprCxr9RLKKzEuIxjZhUoTdfzSuOjBRTYIReIqwxbaypJmOo27Y3tg8RTm5A0gh6EUkTOX/PddHLNpA/mBDPtV+006tMfrrWdFFpNr5NtQQrsnaDw==
1547c8d5-d626-4bcd-86b7-b8dd5a3b2d13	81ee0c62-d2e8-4b83-b02d-da979dae9d94	certificate	MIICmzCCAYMCBgGe4oZYrDANBgkqhkiG9w0BAQsFADARMQ8wDQYDVQQDDAZtYXN0ZXIwHhcNMjYwNjIwMDA1MzM1WhcNMzYwNjIwMDA1NTE1WjARMQ8wDQYDVQQDDAZtYXN0ZXIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDbiYIi/38gzzskVY/XTNBM4D7aZZY5f/PzSyBDzKqCAkZ7oKkksdwwaNAuRGZ5EnZ2Jf7j6FCJ9opomYkC+wrTSVfWtlRxqJGBb0nbgY6p0HdNUUVWplsGB09pwhKGaysdjWrYxE1qawAiunqI1KSl0pvBJ/D3sOTfG7onBYczCe+uPHB9T0Q08MtE6FCP6hN3/j2khgpr+4JeTR687+H4uQWFZlC8CQ6fMQrHj5rOE9BTieeRdOQFxC2FeyaHzNXtpFRg7AyHRj7furyUKafxOm5lhjQwGcWfkDeU/pnwTVgnQ4QREbgbN3jqDcL5mcRMzqqdjHAS5DK8XZfy1lORAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAHxOSOMPh2wipTn8/sjwyh/xRuyO7lPw3CDA81oir1ecjafJVLSKDXfOH/swACPDO/ZAg0tKtK3+hPz9PMMmGebfVjnfv4ttheYj0/mYwT1va/Rj+d2hHbEBgRiIqVNRgrx6GqFDbmX/pb2N8wWasWBBa+UPtuNsXA/tknwxTiFSRZqnu1NV191nU//JeMbCU5xfG3ZLDfdAm3cx4NRntyCXMtIVZYsON2RK/PMggWVSCejtCwYdlkHB6cZJi7kmB0cKUGQz8zXHydmFMqEDlvdXO47D9UdssO2MK3QZvGItoOJn77cxQw0iyJQHEeqoVLkFs2iv334t9KNORw7AJA0=
a6035155-7459-434b-9cb6-70bb2d722a80	81ee0c62-d2e8-4b83-b02d-da979dae9d94	priority	100
e0cec101-9afc-4815-9cd4-52b02010f320	81ee0c62-d2e8-4b83-b02d-da979dae9d94	keyUse	SIG
73d1df08-3c6a-4560-a0e6-e31b03947a28	a8238823-5fd6-42d6-828a-f6677e91e018	kid	1930a87a-206e-46d2-bf44-6c848e263f6a
16576875-f7af-47e7-83e0-4c6876fe40c6	a8238823-5fd6-42d6-828a-f6677e91e018	secret	RAI6M8ifYs0xxzJ_aYY4Ag
ec7e4857-c029-44d2-bdf7-31da33ba0985	a8238823-5fd6-42d6-828a-f6677e91e018	priority	100
\.


--
-- Data for Name: composite_role; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.composite_role (composite, child_role) FROM stdin;
d74b8256-c52a-45ef-86e8-b8ec3391ee89	5c4f9699-e858-4348-8c6e-7ed861b76b81
d74b8256-c52a-45ef-86e8-b8ec3391ee89	ac1db954-21a4-48de-a7fe-e89bc65ebb56
d74b8256-c52a-45ef-86e8-b8ec3391ee89	d8e294ad-eff4-43ab-8031-5839928f0e19
d74b8256-c52a-45ef-86e8-b8ec3391ee89	f78242be-3ae2-4d42-a72e-d287fffad8a7
d74b8256-c52a-45ef-86e8-b8ec3391ee89	f3ab0b2a-359f-439f-8453-5c76ed3f1676
d74b8256-c52a-45ef-86e8-b8ec3391ee89	de03e282-b1a8-40e8-bd0a-f058a4f1e97c
d74b8256-c52a-45ef-86e8-b8ec3391ee89	615bd060-c696-424e-9a22-735dab494777
d74b8256-c52a-45ef-86e8-b8ec3391ee89	3f2be762-9a42-4d49-821c-337339e8e8bc
d74b8256-c52a-45ef-86e8-b8ec3391ee89	d5af39bc-78da-4524-8c9d-8d7bbbf9a82f
d74b8256-c52a-45ef-86e8-b8ec3391ee89	196f1e41-4795-427f-8cdd-422c91be41f9
d74b8256-c52a-45ef-86e8-b8ec3391ee89	71fe12d4-2f50-40f6-baeb-422ed6227bc0
d74b8256-c52a-45ef-86e8-b8ec3391ee89	40069e55-8fb4-428c-86e2-46e097d56085
d74b8256-c52a-45ef-86e8-b8ec3391ee89	adc595c6-3459-4e54-8640-3a32462b46e7
d74b8256-c52a-45ef-86e8-b8ec3391ee89	6e81b2a0-eddc-4532-b510-e7918b84b274
d74b8256-c52a-45ef-86e8-b8ec3391ee89	3bc6395e-80b8-4222-a37f-cc037e766a64
d74b8256-c52a-45ef-86e8-b8ec3391ee89	17efd815-faca-4f67-9c4b-0f85017ce5e8
d74b8256-c52a-45ef-86e8-b8ec3391ee89	e0d52462-d5da-4d44-a134-8b6b7cbfb3be
d74b8256-c52a-45ef-86e8-b8ec3391ee89	5980979e-e8e4-4d2d-a302-2273157922b0
f3ab0b2a-359f-439f-8453-5c76ed3f1676	17efd815-faca-4f67-9c4b-0f85017ce5e8
f78242be-3ae2-4d42-a72e-d287fffad8a7	3bc6395e-80b8-4222-a37f-cc037e766a64
f78242be-3ae2-4d42-a72e-d287fffad8a7	5980979e-e8e4-4d2d-a302-2273157922b0
5f661961-94d9-49fb-9627-50430fff0145	c6fb3296-8dfe-4257-ab78-976e8fa51c33
5f661961-94d9-49fb-9627-50430fff0145	681c4058-b23f-4a15-b885-2ac090faa232
681c4058-b23f-4a15-b885-2ac090faa232	8a8a7751-24ed-486d-84ea-a7ff627d6648
accd35fd-d802-4188-99a1-3040f7987b96	e52928e0-7dc9-4e81-b9d1-aeecd3b0aa92
d74b8256-c52a-45ef-86e8-b8ec3391ee89	b62a30b9-2f46-4b64-8552-8699c982eed0
5f661961-94d9-49fb-9627-50430fff0145	10fbb334-c415-430e-9b09-496038b00760
5f661961-94d9-49fb-9627-50430fff0145	97095622-d040-4d6a-b137-24db9fe89025
\.


--
-- Data for Name: credential; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.credential (id, salt, type, user_id, created_date, user_label, secret_data, credential_data, priority, version) FROM stdin;
95f8fecb-4a9e-45a4-a125-b969dcf128da	\N	password	f35a80a3-4d9f-4a68-997e-c470f30f2281	1781917012859	My password	{"value":"ZJ2FBUdgRpbiKnc5fyso+xl5+oF28CCxtOb8px0wxRY=","salt":"jM1Lzk2NROgNikoynodeBQ==","additionalParameters":{}}	{"hashIterations":5,"algorithm":"argon2","additionalParameters":{"hashLength":["32"],"memory":["7168"],"type":["id"],"version":["1.3"],"parallelism":["1"]}}	10	1
\.


--
-- Data for Name: databasechangelog; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.databasechangelog (id, author, filename, dateexecuted, orderexecuted, exectype, md5sum, description, comments, tag, liquibase, contexts, labels, deployment_id) FROM stdin;
1.0.0.Final-KEYCLOAK-5461	sthorger@redhat.com	META-INF/jpa-changelog-1.0.0.Final.xml	2026-06-20 00:55:10.904561	1	EXECUTED	9:6f1016664e21e16d26517a4418f5e3df	createTable tableName=APPLICATION_DEFAULT_ROLES; createTable tableName=CLIENT; createTable tableName=CLIENT_SESSION; createTable tableName=CLIENT_SESSION_ROLE; createTable tableName=COMPOSITE_ROLE; createTable tableName=CREDENTIAL; createTable tab...		\N	4.33.0	\N	\N	1916909404
1.0.0.Final-KEYCLOAK-5461	sthorger@redhat.com	META-INF/db2-jpa-changelog-1.0.0.Final.xml	2026-06-20 00:55:10.910285	2	MARK_RAN	9:828775b1596a07d1200ba1d49e5e3941	createTable tableName=APPLICATION_DEFAULT_ROLES; createTable tableName=CLIENT; createTable tableName=CLIENT_SESSION; createTable tableName=CLIENT_SESSION_ROLE; createTable tableName=COMPOSITE_ROLE; createTable tableName=CREDENTIAL; createTable tab...		\N	4.33.0	\N	\N	1916909404
1.1.0.Beta1	sthorger@redhat.com	META-INF/jpa-changelog-1.1.0.Beta1.xml	2026-06-20 00:55:10.920064	3	EXECUTED	9:5f090e44a7d595883c1fb61f4b41fd38	delete tableName=CLIENT_SESSION_ROLE; delete tableName=CLIENT_SESSION; delete tableName=USER_SESSION; createTable tableName=CLIENT_ATTRIBUTES; createTable tableName=CLIENT_SESSION_NOTE; createTable tableName=APP_NODE_REGISTRATIONS; addColumn table...		\N	4.33.0	\N	\N	1916909404
1.1.0.Final	sthorger@redhat.com	META-INF/jpa-changelog-1.1.0.Final.xml	2026-06-20 00:55:10.921195	4	EXECUTED	9:c07e577387a3d2c04d1adc9aaad8730e	renameColumn newColumnName=EVENT_TIME, oldColumnName=TIME, tableName=EVENT_ENTITY		\N	4.33.0	\N	\N	1916909404
1.2.0.Beta1	psilva@redhat.com	META-INF/jpa-changelog-1.2.0.Beta1.xml	2026-06-20 00:55:10.945118	5	EXECUTED	9:b68ce996c655922dbcd2fe6b6ae72686	delete tableName=CLIENT_SESSION_ROLE; delete tableName=CLIENT_SESSION_NOTE; delete tableName=CLIENT_SESSION; delete tableName=USER_SESSION; createTable tableName=PROTOCOL_MAPPER; createTable tableName=PROTOCOL_MAPPER_CONFIG; createTable tableName=...		\N	4.33.0	\N	\N	1916909404
1.2.0.Beta1	psilva@redhat.com	META-INF/db2-jpa-changelog-1.2.0.Beta1.xml	2026-06-20 00:55:10.947089	6	MARK_RAN	9:543b5c9989f024fe35c6f6c5a97de88e	delete tableName=CLIENT_SESSION_ROLE; delete tableName=CLIENT_SESSION_NOTE; delete tableName=CLIENT_SESSION; delete tableName=USER_SESSION; createTable tableName=PROTOCOL_MAPPER; createTable tableName=PROTOCOL_MAPPER_CONFIG; createTable tableName=...		\N	4.33.0	\N	\N	1916909404
1.2.0.RC1	bburke@redhat.com	META-INF/jpa-changelog-1.2.0.CR1.xml	2026-06-20 00:55:10.967159	7	EXECUTED	9:765afebbe21cf5bbca048e632df38336	delete tableName=CLIENT_SESSION_ROLE; delete tableName=CLIENT_SESSION_NOTE; delete tableName=CLIENT_SESSION; delete tableName=USER_SESSION_NOTE; delete tableName=USER_SESSION; createTable tableName=MIGRATION_MODEL; createTable tableName=IDENTITY_P...		\N	4.33.0	\N	\N	1916909404
1.2.0.RC1	bburke@redhat.com	META-INF/db2-jpa-changelog-1.2.0.CR1.xml	2026-06-20 00:55:10.969157	8	MARK_RAN	9:db4a145ba11a6fdaefb397f6dbf829a1	delete tableName=CLIENT_SESSION_ROLE; delete tableName=CLIENT_SESSION_NOTE; delete tableName=CLIENT_SESSION; delete tableName=USER_SESSION_NOTE; delete tableName=USER_SESSION; createTable tableName=MIGRATION_MODEL; createTable tableName=IDENTITY_P...		\N	4.33.0	\N	\N	1916909404
1.2.0.Final	keycloak	META-INF/jpa-changelog-1.2.0.Final.xml	2026-06-20 00:55:10.971245	9	EXECUTED	9:9d05c7be10cdb873f8bcb41bc3a8ab23	update tableName=CLIENT; update tableName=CLIENT; update tableName=CLIENT		\N	4.33.0	\N	\N	1916909404
1.3.0	bburke@redhat.com	META-INF/jpa-changelog-1.3.0.xml	2026-06-20 00:55:10.992844	10	EXECUTED	9:18593702353128d53111f9b1ff0b82b8	delete tableName=CLIENT_SESSION_ROLE; delete tableName=CLIENT_SESSION_PROT_MAPPER; delete tableName=CLIENT_SESSION_NOTE; delete tableName=CLIENT_SESSION; delete tableName=USER_SESSION_NOTE; delete tableName=USER_SESSION; createTable tableName=ADMI...		\N	4.33.0	\N	\N	1916909404
1.4.0	bburke@redhat.com	META-INF/jpa-changelog-1.4.0.xml	2026-06-20 00:55:11.005973	11	EXECUTED	9:6122efe5f090e41a85c0f1c9e52cbb62	delete tableName=CLIENT_SESSION_AUTH_STATUS; delete tableName=CLIENT_SESSION_ROLE; delete tableName=CLIENT_SESSION_PROT_MAPPER; delete tableName=CLIENT_SESSION_NOTE; delete tableName=CLIENT_SESSION; delete tableName=USER_SESSION_NOTE; delete table...		\N	4.33.0	\N	\N	1916909404
1.4.0	bburke@redhat.com	META-INF/db2-jpa-changelog-1.4.0.xml	2026-06-20 00:55:11.00754	12	MARK_RAN	9:e1ff28bf7568451453f844c5d54bb0b5	delete tableName=CLIENT_SESSION_AUTH_STATUS; delete tableName=CLIENT_SESSION_ROLE; delete tableName=CLIENT_SESSION_PROT_MAPPER; delete tableName=CLIENT_SESSION_NOTE; delete tableName=CLIENT_SESSION; delete tableName=USER_SESSION_NOTE; delete table...		\N	4.33.0	\N	\N	1916909404
1.5.0	bburke@redhat.com	META-INF/jpa-changelog-1.5.0.xml	2026-06-20 00:55:11.013809	13	EXECUTED	9:7af32cd8957fbc069f796b61217483fd	delete tableName=CLIENT_SESSION_AUTH_STATUS; delete tableName=CLIENT_SESSION_ROLE; delete tableName=CLIENT_SESSION_PROT_MAPPER; delete tableName=CLIENT_SESSION_NOTE; delete tableName=CLIENT_SESSION; delete tableName=USER_SESSION_NOTE; delete table...		\N	4.33.0	\N	\N	1916909404
1.6.1_from15	mposolda@redhat.com	META-INF/jpa-changelog-1.6.1.xml	2026-06-20 00:55:11.017831	14	EXECUTED	9:6005e15e84714cd83226bf7879f54190	addColumn tableName=REALM; addColumn tableName=KEYCLOAK_ROLE; addColumn tableName=CLIENT; createTable tableName=OFFLINE_USER_SESSION; createTable tableName=OFFLINE_CLIENT_SESSION; addPrimaryKey constraintName=CONSTRAINT_OFFL_US_SES_PK2, tableName=...		\N	4.33.0	\N	\N	1916909404
1.6.1_from16-pre	mposolda@redhat.com	META-INF/jpa-changelog-1.6.1.xml	2026-06-20 00:55:11.01835	15	MARK_RAN	9:bf656f5a2b055d07f314431cae76f06c	delete tableName=OFFLINE_CLIENT_SESSION; delete tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
1.6.1_from16	mposolda@redhat.com	META-INF/jpa-changelog-1.6.1.xml	2026-06-20 00:55:11.01919	16	MARK_RAN	9:f8dadc9284440469dcf71e25ca6ab99b	dropPrimaryKey constraintName=CONSTRAINT_OFFLINE_US_SES_PK, tableName=OFFLINE_USER_SESSION; dropPrimaryKey constraintName=CONSTRAINT_OFFLINE_CL_SES_PK, tableName=OFFLINE_CLIENT_SESSION; addColumn tableName=OFFLINE_USER_SESSION; update tableName=OF...		\N	4.33.0	\N	\N	1916909404
1.6.1	mposolda@redhat.com	META-INF/jpa-changelog-1.6.1.xml	2026-06-20 00:55:11.020008	17	EXECUTED	9:d41d8cd98f00b204e9800998ecf8427e	empty		\N	4.33.0	\N	\N	1916909404
1.7.0	bburke@redhat.com	META-INF/jpa-changelog-1.7.0.xml	2026-06-20 00:55:11.029582	18	EXECUTED	9:3368ff0be4c2855ee2dd9ca813b38d8e	createTable tableName=KEYCLOAK_GROUP; createTable tableName=GROUP_ROLE_MAPPING; createTable tableName=GROUP_ATTRIBUTE; createTable tableName=USER_GROUP_MEMBERSHIP; createTable tableName=REALM_DEFAULT_GROUPS; addColumn tableName=IDENTITY_PROVIDER; ...		\N	4.33.0	\N	\N	1916909404
1.8.0	mposolda@redhat.com	META-INF/jpa-changelog-1.8.0.xml	2026-06-20 00:55:11.039072	19	EXECUTED	9:8ac2fb5dd030b24c0570a763ed75ed20	addColumn tableName=IDENTITY_PROVIDER; createTable tableName=CLIENT_TEMPLATE; createTable tableName=CLIENT_TEMPLATE_ATTRIBUTES; createTable tableName=TEMPLATE_SCOPE_MAPPING; dropNotNullConstraint columnName=CLIENT_ID, tableName=PROTOCOL_MAPPER; ad...		\N	4.33.0	\N	\N	1916909404
1.8.0-2	keycloak	META-INF/jpa-changelog-1.8.0.xml	2026-06-20 00:55:11.040259	20	EXECUTED	9:f91ddca9b19743db60e3057679810e6c	dropDefaultValue columnName=ALGORITHM, tableName=CREDENTIAL; update tableName=CREDENTIAL		\N	4.33.0	\N	\N	1916909404
22.0.5-24031	keycloak	META-INF/jpa-changelog-22.0.0.xml	2026-06-20 00:55:12.053183	119	MARK_RAN	9:a60d2d7b315ec2d3eba9e2f145f9df28	customChange		\N	4.33.0	\N	\N	1916909404
1.8.0	mposolda@redhat.com	META-INF/db2-jpa-changelog-1.8.0.xml	2026-06-20 00:55:11.041488	21	MARK_RAN	9:831e82914316dc8a57dc09d755f23c51	addColumn tableName=IDENTITY_PROVIDER; createTable tableName=CLIENT_TEMPLATE; createTable tableName=CLIENT_TEMPLATE_ATTRIBUTES; createTable tableName=TEMPLATE_SCOPE_MAPPING; dropNotNullConstraint columnName=CLIENT_ID, tableName=PROTOCOL_MAPPER; ad...		\N	4.33.0	\N	\N	1916909404
1.8.0-2	keycloak	META-INF/db2-jpa-changelog-1.8.0.xml	2026-06-20 00:55:11.042421	22	MARK_RAN	9:f91ddca9b19743db60e3057679810e6c	dropDefaultValue columnName=ALGORITHM, tableName=CREDENTIAL; update tableName=CREDENTIAL		\N	4.33.0	\N	\N	1916909404
1.9.0	mposolda@redhat.com	META-INF/jpa-changelog-1.9.0.xml	2026-06-20 00:55:11.063584	23	EXECUTED	9:bc3d0f9e823a69dc21e23e94c7a94bb1	update tableName=REALM; update tableName=REALM; update tableName=REALM; update tableName=REALM; update tableName=CREDENTIAL; update tableName=CREDENTIAL; update tableName=CREDENTIAL; update tableName=REALM; update tableName=REALM; customChange; dr...		\N	4.33.0	\N	\N	1916909404
1.9.1	keycloak	META-INF/jpa-changelog-1.9.1.xml	2026-06-20 00:55:11.065276	24	EXECUTED	9:c9999da42f543575ab790e76439a2679	modifyDataType columnName=PRIVATE_KEY, tableName=REALM; modifyDataType columnName=PUBLIC_KEY, tableName=REALM; modifyDataType columnName=CERTIFICATE, tableName=REALM		\N	4.33.0	\N	\N	1916909404
1.9.1	keycloak	META-INF/db2-jpa-changelog-1.9.1.xml	2026-06-20 00:55:11.065759	25	MARK_RAN	9:0d6c65c6f58732d81569e77b10ba301d	modifyDataType columnName=PRIVATE_KEY, tableName=REALM; modifyDataType columnName=CERTIFICATE, tableName=REALM		\N	4.33.0	\N	\N	1916909404
1.9.2	keycloak	META-INF/jpa-changelog-1.9.2.xml	2026-06-20 00:55:11.153118	26	EXECUTED	9:fc576660fc016ae53d2d4778d84d86d0	createIndex indexName=IDX_USER_EMAIL, tableName=USER_ENTITY; createIndex indexName=IDX_USER_ROLE_MAPPING, tableName=USER_ROLE_MAPPING; createIndex indexName=IDX_USER_GROUP_MAPPING, tableName=USER_GROUP_MEMBERSHIP; createIndex indexName=IDX_USER_CO...		\N	4.33.0	\N	\N	1916909404
authz-2.0.0	psilva@redhat.com	META-INF/jpa-changelog-authz-2.0.0.xml	2026-06-20 00:55:11.165845	27	EXECUTED	9:43ed6b0da89ff77206289e87eaa9c024	createTable tableName=RESOURCE_SERVER; addPrimaryKey constraintName=CONSTRAINT_FARS, tableName=RESOURCE_SERVER; addUniqueConstraint constraintName=UK_AU8TT6T700S9V50BU18WS5HA6, tableName=RESOURCE_SERVER; createTable tableName=RESOURCE_SERVER_RESOU...		\N	4.33.0	\N	\N	1916909404
authz-2.5.1	psilva@redhat.com	META-INF/jpa-changelog-authz-2.5.1.xml	2026-06-20 00:55:11.166937	28	EXECUTED	9:44bae577f551b3738740281eceb4ea70	update tableName=RESOURCE_SERVER_POLICY		\N	4.33.0	\N	\N	1916909404
2.1.0-KEYCLOAK-5461	bburke@redhat.com	META-INF/jpa-changelog-2.1.0.xml	2026-06-20 00:55:11.175779	29	EXECUTED	9:bd88e1f833df0420b01e114533aee5e8	createTable tableName=BROKER_LINK; createTable tableName=FED_USER_ATTRIBUTE; createTable tableName=FED_USER_CONSENT; createTable tableName=FED_USER_CONSENT_ROLE; createTable tableName=FED_USER_CONSENT_PROT_MAPPER; createTable tableName=FED_USER_CR...		\N	4.33.0	\N	\N	1916909404
2.2.0	bburke@redhat.com	META-INF/jpa-changelog-2.2.0.xml	2026-06-20 00:55:11.1786	30	EXECUTED	9:a7022af5267f019d020edfe316ef4371	addColumn tableName=ADMIN_EVENT_ENTITY; createTable tableName=CREDENTIAL_ATTRIBUTE; createTable tableName=FED_CREDENTIAL_ATTRIBUTE; modifyDataType columnName=VALUE, tableName=CREDENTIAL; addForeignKeyConstraint baseTableName=FED_CREDENTIAL_ATTRIBU...		\N	4.33.0	\N	\N	1916909404
2.3.0	bburke@redhat.com	META-INF/jpa-changelog-2.3.0.xml	2026-06-20 00:55:11.182799	31	EXECUTED	9:fc155c394040654d6a79227e56f5e25a	createTable tableName=FEDERATED_USER; addPrimaryKey constraintName=CONSTR_FEDERATED_USER, tableName=FEDERATED_USER; dropDefaultValue columnName=TOTP, tableName=USER_ENTITY; dropColumn columnName=TOTP, tableName=USER_ENTITY; addColumn tableName=IDE...		\N	4.33.0	\N	\N	1916909404
2.4.0	bburke@redhat.com	META-INF/jpa-changelog-2.4.0.xml	2026-06-20 00:55:11.184002	32	EXECUTED	9:eac4ffb2a14795e5dc7b426063e54d88	customChange		\N	4.33.0	\N	\N	1916909404
2.5.0	bburke@redhat.com	META-INF/jpa-changelog-2.5.0.xml	2026-06-20 00:55:11.185367	33	EXECUTED	9:54937c05672568c4c64fc9524c1e9462	customChange; modifyDataType columnName=USER_ID, tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
2.5.0-unicode-oracle	hmlnarik@redhat.com	META-INF/jpa-changelog-2.5.0.xml	2026-06-20 00:55:11.185955	34	MARK_RAN	9:f9753208029f582525ed12011a19d054	modifyDataType columnName=DESCRIPTION, tableName=AUTHENTICATION_FLOW; modifyDataType columnName=DESCRIPTION, tableName=CLIENT_TEMPLATE; modifyDataType columnName=DESCRIPTION, tableName=RESOURCE_SERVER_POLICY; modifyDataType columnName=DESCRIPTION,...		\N	4.33.0	\N	\N	1916909404
2.5.0-unicode-other-dbs	hmlnarik@redhat.com	META-INF/jpa-changelog-2.5.0.xml	2026-06-20 00:55:11.191409	35	EXECUTED	9:33d72168746f81f98ae3a1e8e0ca3554	modifyDataType columnName=DESCRIPTION, tableName=AUTHENTICATION_FLOW; modifyDataType columnName=DESCRIPTION, tableName=CLIENT_TEMPLATE; modifyDataType columnName=DESCRIPTION, tableName=RESOURCE_SERVER_POLICY; modifyDataType columnName=DESCRIPTION,...		\N	4.33.0	\N	\N	1916909404
2.5.0-duplicate-email-support	slawomir@dabek.name	META-INF/jpa-changelog-2.5.0.xml	2026-06-20 00:55:11.193059	36	EXECUTED	9:61b6d3d7a4c0e0024b0c839da283da0c	addColumn tableName=REALM		\N	4.33.0	\N	\N	1916909404
2.5.0-unique-group-names	hmlnarik@redhat.com	META-INF/jpa-changelog-2.5.0.xml	2026-06-20 00:55:11.194041	37	EXECUTED	9:8dcac7bdf7378e7d823cdfddebf72fda	addUniqueConstraint constraintName=SIBLING_NAMES, tableName=KEYCLOAK_GROUP		\N	4.33.0	\N	\N	1916909404
2.5.1	bburke@redhat.com	META-INF/jpa-changelog-2.5.1.xml	2026-06-20 00:55:11.194925	38	EXECUTED	9:a2b870802540cb3faa72098db5388af3	addColumn tableName=FED_USER_CONSENT		\N	4.33.0	\N	\N	1916909404
3.0.0	bburke@redhat.com	META-INF/jpa-changelog-3.0.0.xml	2026-06-20 00:55:11.195703	39	EXECUTED	9:132a67499ba24bcc54fb5cbdcfe7e4c0	addColumn tableName=IDENTITY_PROVIDER		\N	4.33.0	\N	\N	1916909404
3.2.0-fix	keycloak	META-INF/jpa-changelog-3.2.0.xml	2026-06-20 00:55:11.196019	40	MARK_RAN	9:938f894c032f5430f2b0fafb1a243462	addNotNullConstraint columnName=REALM_ID, tableName=CLIENT_INITIAL_ACCESS		\N	4.33.0	\N	\N	1916909404
3.2.0-fix-with-keycloak-5416	keycloak	META-INF/jpa-changelog-3.2.0.xml	2026-06-20 00:55:11.196704	41	MARK_RAN	9:845c332ff1874dc5d35974b0babf3006	dropIndex indexName=IDX_CLIENT_INIT_ACC_REALM, tableName=CLIENT_INITIAL_ACCESS; addNotNullConstraint columnName=REALM_ID, tableName=CLIENT_INITIAL_ACCESS; createIndex indexName=IDX_CLIENT_INIT_ACC_REALM, tableName=CLIENT_INITIAL_ACCESS		\N	4.33.0	\N	\N	1916909404
3.2.0-fix-offline-sessions	hmlnarik	META-INF/jpa-changelog-3.2.0.xml	2026-06-20 00:55:11.198104	42	EXECUTED	9:fc86359c079781adc577c5a217e4d04c	customChange		\N	4.33.0	\N	\N	1916909404
3.2.0-fixed	keycloak	META-INF/jpa-changelog-3.2.0.xml	2026-06-20 00:55:11.574991	43	EXECUTED	9:59a64800e3c0d09b825f8a3b444fa8f4	addColumn tableName=REALM; dropPrimaryKey constraintName=CONSTRAINT_OFFL_CL_SES_PK2, tableName=OFFLINE_CLIENT_SESSION; dropColumn columnName=CLIENT_SESSION_ID, tableName=OFFLINE_CLIENT_SESSION; addPrimaryKey constraintName=CONSTRAINT_OFFL_CL_SES_P...		\N	4.33.0	\N	\N	1916909404
3.3.0	keycloak	META-INF/jpa-changelog-3.3.0.xml	2026-06-20 00:55:11.576426	44	EXECUTED	9:d48d6da5c6ccf667807f633fe489ce88	addColumn tableName=USER_ENTITY		\N	4.33.0	\N	\N	1916909404
authz-3.4.0.CR1-resource-server-pk-change-part1	glavoie@gmail.com	META-INF/jpa-changelog-authz-3.4.0.CR1.xml	2026-06-20 00:55:11.577393	45	EXECUTED	9:dde36f7973e80d71fceee683bc5d2951	addColumn tableName=RESOURCE_SERVER_POLICY; addColumn tableName=RESOURCE_SERVER_RESOURCE; addColumn tableName=RESOURCE_SERVER_SCOPE		\N	4.33.0	\N	\N	1916909404
authz-3.4.0.CR1-resource-server-pk-change-part2-KEYCLOAK-6095	hmlnarik@redhat.com	META-INF/jpa-changelog-authz-3.4.0.CR1.xml	2026-06-20 00:55:11.578497	46	EXECUTED	9:b855e9b0a406b34fa323235a0cf4f640	customChange		\N	4.33.0	\N	\N	1916909404
authz-3.4.0.CR1-resource-server-pk-change-part3-fixed	glavoie@gmail.com	META-INF/jpa-changelog-authz-3.4.0.CR1.xml	2026-06-20 00:55:11.578823	47	MARK_RAN	9:51abbacd7b416c50c4421a8cabf7927e	dropIndex indexName=IDX_RES_SERV_POL_RES_SERV, tableName=RESOURCE_SERVER_POLICY; dropIndex indexName=IDX_RES_SRV_RES_RES_SRV, tableName=RESOURCE_SERVER_RESOURCE; dropIndex indexName=IDX_RES_SRV_SCOPE_RES_SRV, tableName=RESOURCE_SERVER_SCOPE		\N	4.33.0	\N	\N	1916909404
authz-3.4.0.CR1-resource-server-pk-change-part3-fixed-nodropindex	glavoie@gmail.com	META-INF/jpa-changelog-authz-3.4.0.CR1.xml	2026-06-20 00:55:11.610203	48	EXECUTED	9:bdc99e567b3398bac83263d375aad143	addNotNullConstraint columnName=RESOURCE_SERVER_CLIENT_ID, tableName=RESOURCE_SERVER_POLICY; addNotNullConstraint columnName=RESOURCE_SERVER_CLIENT_ID, tableName=RESOURCE_SERVER_RESOURCE; addNotNullConstraint columnName=RESOURCE_SERVER_CLIENT_ID, ...		\N	4.33.0	\N	\N	1916909404
authn-3.4.0.CR1-refresh-token-max-reuse	glavoie@gmail.com	META-INF/jpa-changelog-authz-3.4.0.CR1.xml	2026-06-20 00:55:11.61161	49	EXECUTED	9:d198654156881c46bfba39abd7769e69	addColumn tableName=REALM		\N	4.33.0	\N	\N	1916909404
3.4.0	keycloak	META-INF/jpa-changelog-3.4.0.xml	2026-06-20 00:55:11.616581	50	EXECUTED	9:cfdd8736332ccdd72c5256ccb42335db	addPrimaryKey constraintName=CONSTRAINT_REALM_DEFAULT_ROLES, tableName=REALM_DEFAULT_ROLES; addPrimaryKey constraintName=CONSTRAINT_COMPOSITE_ROLE, tableName=COMPOSITE_ROLE; addPrimaryKey constraintName=CONSTR_REALM_DEFAULT_GROUPS, tableName=REALM...		\N	4.33.0	\N	\N	1916909404
3.4.0-KEYCLOAK-5230	hmlnarik@redhat.com	META-INF/jpa-changelog-3.4.0.xml	2026-06-20 00:55:11.704349	51	EXECUTED	9:7c84de3d9bd84d7f077607c1a4dcb714	createIndex indexName=IDX_FU_ATTRIBUTE, tableName=FED_USER_ATTRIBUTE; createIndex indexName=IDX_FU_CONSENT, tableName=FED_USER_CONSENT; createIndex indexName=IDX_FU_CONSENT_RU, tableName=FED_USER_CONSENT; createIndex indexName=IDX_FU_CREDENTIAL, t...		\N	4.33.0	\N	\N	1916909404
3.4.1	psilva@redhat.com	META-INF/jpa-changelog-3.4.1.xml	2026-06-20 00:55:11.705218	52	EXECUTED	9:5a6bb36cbefb6a9d6928452c0852af2d	modifyDataType columnName=VALUE, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
3.4.2	keycloak	META-INF/jpa-changelog-3.4.2.xml	2026-06-20 00:55:11.706358	53	EXECUTED	9:8f23e334dbc59f82e0a328373ca6ced0	update tableName=REALM		\N	4.33.0	\N	\N	1916909404
3.4.2-KEYCLOAK-5172	mkanis@redhat.com	META-INF/jpa-changelog-3.4.2.xml	2026-06-20 00:55:11.707135	54	EXECUTED	9:9156214268f09d970cdf0e1564d866af	update tableName=CLIENT		\N	4.33.0	\N	\N	1916909404
4.0.0-KEYCLOAK-6335	bburke@redhat.com	META-INF/jpa-changelog-4.0.0.xml	2026-06-20 00:55:11.708296	55	EXECUTED	9:db806613b1ed154826c02610b7dbdf74	createTable tableName=CLIENT_AUTH_FLOW_BINDINGS; addPrimaryKey constraintName=C_CLI_FLOW_BIND, tableName=CLIENT_AUTH_FLOW_BINDINGS		\N	4.33.0	\N	\N	1916909404
4.0.0-CLEANUP-UNUSED-TABLE	bburke@redhat.com	META-INF/jpa-changelog-4.0.0.xml	2026-06-20 00:55:11.709273	56	EXECUTED	9:229a041fb72d5beac76bb94a5fa709de	dropTable tableName=CLIENT_IDENTITY_PROV_MAPPING		\N	4.33.0	\N	\N	1916909404
4.0.0-KEYCLOAK-6228	bburke@redhat.com	META-INF/jpa-changelog-4.0.0.xml	2026-06-20 00:55:11.72127	57	EXECUTED	9:079899dade9c1e683f26b2aa9ca6ff04	dropUniqueConstraint constraintName=UK_JKUWUVD56ONTGSUHOGM8UEWRT, tableName=USER_CONSENT; dropNotNullConstraint columnName=CLIENT_ID, tableName=USER_CONSENT; addColumn tableName=USER_CONSENT; addUniqueConstraint constraintName=UK_JKUWUVD56ONTGSUHO...		\N	4.33.0	\N	\N	1916909404
4.0.0-KEYCLOAK-5579-fixed	mposolda@redhat.com	META-INF/jpa-changelog-4.0.0.xml	2026-06-20 00:55:11.822753	58	EXECUTED	9:139b79bcbbfe903bb1c2d2a4dbf001d9	dropForeignKeyConstraint baseTableName=CLIENT_TEMPLATE_ATTRIBUTES, constraintName=FK_CL_TEMPL_ATTR_TEMPL; renameTable newTableName=CLIENT_SCOPE_ATTRIBUTES, oldTableName=CLIENT_TEMPLATE_ATTRIBUTES; renameColumn newColumnName=SCOPE_ID, oldColumnName...		\N	4.33.0	\N	\N	1916909404
authz-4.0.0.CR1	psilva@redhat.com	META-INF/jpa-changelog-authz-4.0.0.CR1.xml	2026-06-20 00:55:11.827939	59	EXECUTED	9:b55738ad889860c625ba2bf483495a04	createTable tableName=RESOURCE_SERVER_PERM_TICKET; addPrimaryKey constraintName=CONSTRAINT_FAPMT, tableName=RESOURCE_SERVER_PERM_TICKET; addForeignKeyConstraint baseTableName=RESOURCE_SERVER_PERM_TICKET, constraintName=FK_FRSRHO213XCX4WNKOG82SSPMT...		\N	4.33.0	\N	\N	1916909404
authz-4.0.0.Beta3	psilva@redhat.com	META-INF/jpa-changelog-authz-4.0.0.Beta3.xml	2026-06-20 00:55:11.829438	60	EXECUTED	9:e0057eac39aa8fc8e09ac6cfa4ae15fe	addColumn tableName=RESOURCE_SERVER_POLICY; addColumn tableName=RESOURCE_SERVER_PERM_TICKET; addForeignKeyConstraint baseTableName=RESOURCE_SERVER_PERM_TICKET, constraintName=FK_FRSRPO2128CX4WNKOG82SSRFY, referencedTableName=RESOURCE_SERVER_POLICY		\N	4.33.0	\N	\N	1916909404
authz-4.2.0.Final	mhajas@redhat.com	META-INF/jpa-changelog-authz-4.2.0.Final.xml	2026-06-20 00:55:11.831196	61	EXECUTED	9:42a33806f3a0443fe0e7feeec821326c	createTable tableName=RESOURCE_URIS; addForeignKeyConstraint baseTableName=RESOURCE_URIS, constraintName=FK_RESOURCE_SERVER_URIS, referencedTableName=RESOURCE_SERVER_RESOURCE; customChange; dropColumn columnName=URI, tableName=RESOURCE_SERVER_RESO...		\N	4.33.0	\N	\N	1916909404
authz-4.2.0.Final-KEYCLOAK-9944	hmlnarik@redhat.com	META-INF/jpa-changelog-authz-4.2.0.Final.xml	2026-06-20 00:55:11.831907	62	EXECUTED	9:9968206fca46eecc1f51db9c024bfe56	addPrimaryKey constraintName=CONSTRAINT_RESOUR_URIS_PK, tableName=RESOURCE_URIS		\N	4.33.0	\N	\N	1916909404
4.2.0-KEYCLOAK-6313	wadahiro@gmail.com	META-INF/jpa-changelog-4.2.0.xml	2026-06-20 00:55:11.832609	63	EXECUTED	9:92143a6daea0a3f3b8f598c97ce55c3d	addColumn tableName=REQUIRED_ACTION_PROVIDER		\N	4.33.0	\N	\N	1916909404
4.3.0-KEYCLOAK-7984	wadahiro@gmail.com	META-INF/jpa-changelog-4.3.0.xml	2026-06-20 00:55:11.833292	64	EXECUTED	9:82bab26a27195d889fb0429003b18f40	update tableName=REQUIRED_ACTION_PROVIDER		\N	4.33.0	\N	\N	1916909404
4.6.0-KEYCLOAK-7950	psilva@redhat.com	META-INF/jpa-changelog-4.6.0.xml	2026-06-20 00:55:11.834491	65	EXECUTED	9:e590c88ddc0b38b0ae4249bbfcb5abc3	update tableName=RESOURCE_SERVER_RESOURCE		\N	4.33.0	\N	\N	1916909404
4.6.0-KEYCLOAK-8377	keycloak	META-INF/jpa-changelog-4.6.0.xml	2026-06-20 00:55:11.847083	66	EXECUTED	9:5c1f475536118dbdc38d5d7977950cc0	createTable tableName=ROLE_ATTRIBUTE; addPrimaryKey constraintName=CONSTRAINT_ROLE_ATTRIBUTE_PK, tableName=ROLE_ATTRIBUTE; addForeignKeyConstraint baseTableName=ROLE_ATTRIBUTE, constraintName=FK_ROLE_ATTRIBUTE_ID, referencedTableName=KEYCLOAK_ROLE...		\N	4.33.0	\N	\N	1916909404
4.6.0-KEYCLOAK-8555	gideonray@gmail.com	META-INF/jpa-changelog-4.6.0.xml	2026-06-20 00:55:11.856412	67	EXECUTED	9:e7c9f5f9c4d67ccbbcc215440c718a17	createIndex indexName=IDX_COMPONENT_PROVIDER_TYPE, tableName=COMPONENT		\N	4.33.0	\N	\N	1916909404
4.7.0-KEYCLOAK-1267	sguilhen@redhat.com	META-INF/jpa-changelog-4.7.0.xml	2026-06-20 00:55:11.857584	68	EXECUTED	9:88e0bfdda924690d6f4e430c53447dd5	addColumn tableName=REALM		\N	4.33.0	\N	\N	1916909404
4.7.0-KEYCLOAK-7275	keycloak	META-INF/jpa-changelog-4.7.0.xml	2026-06-20 00:55:11.868286	69	EXECUTED	9:f53177f137e1c46b6a88c59ec1cb5218	renameColumn newColumnName=CREATED_ON, oldColumnName=LAST_SESSION_REFRESH, tableName=OFFLINE_USER_SESSION; addNotNullConstraint columnName=CREATED_ON, tableName=OFFLINE_USER_SESSION; addColumn tableName=OFFLINE_USER_SESSION; customChange; createIn...		\N	4.33.0	\N	\N	1916909404
4.8.0-KEYCLOAK-8835	sguilhen@redhat.com	META-INF/jpa-changelog-4.8.0.xml	2026-06-20 00:55:11.869754	70	EXECUTED	9:a74d33da4dc42a37ec27121580d1459f	addNotNullConstraint columnName=SSO_MAX_LIFESPAN_REMEMBER_ME, tableName=REALM; addNotNullConstraint columnName=SSO_IDLE_TIMEOUT_REMEMBER_ME, tableName=REALM		\N	4.33.0	\N	\N	1916909404
authz-7.0.0-KEYCLOAK-10443	psilva@redhat.com	META-INF/jpa-changelog-authz-7.0.0.xml	2026-06-20 00:55:11.870633	71	EXECUTED	9:fd4ade7b90c3b67fae0bfcfcb42dfb5f	addColumn tableName=RESOURCE_SERVER		\N	4.33.0	\N	\N	1916909404
8.0.0-adding-credential-columns	keycloak	META-INF/jpa-changelog-8.0.0.xml	2026-06-20 00:55:11.872573	72	EXECUTED	9:aa072ad090bbba210d8f18781b8cebf4	addColumn tableName=CREDENTIAL; addColumn tableName=FED_USER_CREDENTIAL		\N	4.33.0	\N	\N	1916909404
8.0.0-updating-credential-data-not-oracle-fixed	keycloak	META-INF/jpa-changelog-8.0.0.xml	2026-06-20 00:55:11.875035	73	EXECUTED	9:1ae6be29bab7c2aa376f6983b932be37	update tableName=CREDENTIAL; update tableName=CREDENTIAL; update tableName=CREDENTIAL; update tableName=FED_USER_CREDENTIAL; update tableName=FED_USER_CREDENTIAL; update tableName=FED_USER_CREDENTIAL		\N	4.33.0	\N	\N	1916909404
8.0.0-updating-credential-data-oracle-fixed	keycloak	META-INF/jpa-changelog-8.0.0.xml	2026-06-20 00:55:11.875679	74	MARK_RAN	9:14706f286953fc9a25286dbd8fb30d97	update tableName=CREDENTIAL; update tableName=CREDENTIAL; update tableName=CREDENTIAL; update tableName=FED_USER_CREDENTIAL; update tableName=FED_USER_CREDENTIAL; update tableName=FED_USER_CREDENTIAL		\N	4.33.0	\N	\N	1916909404
8.0.0-credential-cleanup-fixed	keycloak	META-INF/jpa-changelog-8.0.0.xml	2026-06-20 00:55:11.88082	75	EXECUTED	9:2b9cc12779be32c5b40e2e67711a218b	dropDefaultValue columnName=COUNTER, tableName=CREDENTIAL; dropDefaultValue columnName=DIGITS, tableName=CREDENTIAL; dropDefaultValue columnName=PERIOD, tableName=CREDENTIAL; dropDefaultValue columnName=ALGORITHM, tableName=CREDENTIAL; dropColumn ...		\N	4.33.0	\N	\N	1916909404
8.0.0-resource-tag-support	keycloak	META-INF/jpa-changelog-8.0.0.xml	2026-06-20 00:55:11.890925	76	EXECUTED	9:91fa186ce7a5af127a2d7a91ee083cc5	addColumn tableName=MIGRATION_MODEL; createIndex indexName=IDX_UPDATE_TIME, tableName=MIGRATION_MODEL		\N	4.33.0	\N	\N	1916909404
9.0.0-always-display-client	keycloak	META-INF/jpa-changelog-9.0.0.xml	2026-06-20 00:55:11.891972	77	EXECUTED	9:6335e5c94e83a2639ccd68dd24e2e5ad	addColumn tableName=CLIENT		\N	4.33.0	\N	\N	1916909404
9.0.0-drop-constraints-for-column-increase	keycloak	META-INF/jpa-changelog-9.0.0.xml	2026-06-20 00:55:11.892296	78	MARK_RAN	9:6bdb5658951e028bfe16fa0a8228b530	dropUniqueConstraint constraintName=UK_FRSR6T700S9V50BU18WS5PMT, tableName=RESOURCE_SERVER_PERM_TICKET; dropUniqueConstraint constraintName=UK_FRSR6T700S9V50BU18WS5HA6, tableName=RESOURCE_SERVER_RESOURCE; dropPrimaryKey constraintName=CONSTRAINT_O...		\N	4.33.0	\N	\N	1916909404
9.0.0-increase-column-size-federated-fk	keycloak	META-INF/jpa-changelog-9.0.0.xml	2026-06-20 00:55:11.896147	79	EXECUTED	9:d5bc15a64117ccad481ce8792d4c608f	modifyDataType columnName=CLIENT_ID, tableName=FED_USER_CONSENT; modifyDataType columnName=CLIENT_REALM_CONSTRAINT, tableName=KEYCLOAK_ROLE; modifyDataType columnName=OWNER, tableName=RESOURCE_SERVER_POLICY; modifyDataType columnName=CLIENT_ID, ta...		\N	4.33.0	\N	\N	1916909404
9.0.0-recreate-constraints-after-column-increase	keycloak	META-INF/jpa-changelog-9.0.0.xml	2026-06-20 00:55:11.896579	80	MARK_RAN	9:077cba51999515f4d3e7ad5619ab592c	addNotNullConstraint columnName=CLIENT_ID, tableName=OFFLINE_CLIENT_SESSION; addNotNullConstraint columnName=OWNER, tableName=RESOURCE_SERVER_PERM_TICKET; addNotNullConstraint columnName=REQUESTER, tableName=RESOURCE_SERVER_PERM_TICKET; addNotNull...		\N	4.33.0	\N	\N	1916909404
9.0.1-add-index-to-client.client_id	keycloak	META-INF/jpa-changelog-9.0.1.xml	2026-06-20 00:55:11.906369	81	EXECUTED	9:be969f08a163bf47c6b9e9ead8ac2afb	createIndex indexName=IDX_CLIENT_ID, tableName=CLIENT		\N	4.33.0	\N	\N	1916909404
9.0.1-KEYCLOAK-12579-drop-constraints	keycloak	META-INF/jpa-changelog-9.0.1.xml	2026-06-20 00:55:11.906728	82	MARK_RAN	9:6d3bb4408ba5a72f39bd8a0b301ec6e3	dropUniqueConstraint constraintName=SIBLING_NAMES, tableName=KEYCLOAK_GROUP		\N	4.33.0	\N	\N	1916909404
9.0.1-KEYCLOAK-12579-add-not-null-constraint	keycloak	META-INF/jpa-changelog-9.0.1.xml	2026-06-20 00:55:11.907721	83	EXECUTED	9:966bda61e46bebf3cc39518fbed52fa7	addNotNullConstraint columnName=PARENT_GROUP, tableName=KEYCLOAK_GROUP		\N	4.33.0	\N	\N	1916909404
9.0.1-KEYCLOAK-12579-recreate-constraints	keycloak	META-INF/jpa-changelog-9.0.1.xml	2026-06-20 00:55:11.908205	84	MARK_RAN	9:8dcac7bdf7378e7d823cdfddebf72fda	addUniqueConstraint constraintName=SIBLING_NAMES, tableName=KEYCLOAK_GROUP		\N	4.33.0	\N	\N	1916909404
9.0.1-add-index-to-events	keycloak	META-INF/jpa-changelog-9.0.1.xml	2026-06-20 00:55:11.917281	85	EXECUTED	9:7d93d602352a30c0c317e6a609b56599	createIndex indexName=IDX_EVENT_TIME, tableName=EVENT_ENTITY		\N	4.33.0	\N	\N	1916909404
map-remove-ri	keycloak	META-INF/jpa-changelog-11.0.0.xml	2026-06-20 00:55:11.918444	86	EXECUTED	9:71c5969e6cdd8d7b6f47cebc86d37627	dropForeignKeyConstraint baseTableName=REALM, constraintName=FK_TRAF444KK6QRKMS7N56AIWQ5Y; dropForeignKeyConstraint baseTableName=KEYCLOAK_ROLE, constraintName=FK_KJHO5LE2C0RAL09FL8CM9WFW9		\N	4.33.0	\N	\N	1916909404
map-remove-ri	keycloak	META-INF/jpa-changelog-12.0.0.xml	2026-06-20 00:55:11.920304	87	EXECUTED	9:a9ba7d47f065f041b7da856a81762021	dropForeignKeyConstraint baseTableName=REALM_DEFAULT_GROUPS, constraintName=FK_DEF_GROUPS_GROUP; dropForeignKeyConstraint baseTableName=REALM_DEFAULT_ROLES, constraintName=FK_H4WPD7W4HSOOLNI3H0SW7BTJE; dropForeignKeyConstraint baseTableName=CLIENT...		\N	4.33.0	\N	\N	1916909404
12.1.0-add-realm-localization-table	keycloak	META-INF/jpa-changelog-12.0.0.xml	2026-06-20 00:55:11.921594	88	EXECUTED	9:fffabce2bc01e1a8f5110d5278500065	createTable tableName=REALM_LOCALIZATIONS; addPrimaryKey tableName=REALM_LOCALIZATIONS		\N	4.33.0	\N	\N	1916909404
default-roles	keycloak	META-INF/jpa-changelog-13.0.0.xml	2026-06-20 00:55:11.923452	89	EXECUTED	9:fa8a5b5445e3857f4b010bafb5009957	addColumn tableName=REALM; customChange		\N	4.33.0	\N	\N	1916909404
default-roles-cleanup	keycloak	META-INF/jpa-changelog-13.0.0.xml	2026-06-20 00:55:11.925052	90	EXECUTED	9:67ac3241df9a8582d591c5ed87125f39	dropTable tableName=REALM_DEFAULT_ROLES; dropTable tableName=CLIENT_DEFAULT_ROLES		\N	4.33.0	\N	\N	1916909404
13.0.0-KEYCLOAK-16844	keycloak	META-INF/jpa-changelog-13.0.0.xml	2026-06-20 00:55:11.935183	91	EXECUTED	9:ad1194d66c937e3ffc82386c050ba089	createIndex indexName=IDX_OFFLINE_USS_PRELOAD, tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
map-remove-ri-13.0.0	keycloak	META-INF/jpa-changelog-13.0.0.xml	2026-06-20 00:55:11.937372	92	EXECUTED	9:d9be619d94af5a2f5d07b9f003543b91	dropForeignKeyConstraint baseTableName=DEFAULT_CLIENT_SCOPE, constraintName=FK_R_DEF_CLI_SCOPE_SCOPE; dropForeignKeyConstraint baseTableName=CLIENT_SCOPE_CLIENT, constraintName=FK_C_CLI_SCOPE_SCOPE; dropForeignKeyConstraint baseTableName=CLIENT_SC...		\N	4.33.0	\N	\N	1916909404
13.0.0-KEYCLOAK-17992-drop-constraints	keycloak	META-INF/jpa-changelog-13.0.0.xml	2026-06-20 00:55:11.937728	93	MARK_RAN	9:544d201116a0fcc5a5da0925fbbc3bde	dropPrimaryKey constraintName=C_CLI_SCOPE_BIND, tableName=CLIENT_SCOPE_CLIENT; dropIndex indexName=IDX_CLSCOPE_CL, tableName=CLIENT_SCOPE_CLIENT; dropIndex indexName=IDX_CL_CLSCOPE, tableName=CLIENT_SCOPE_CLIENT		\N	4.33.0	\N	\N	1916909404
13.0.0-increase-column-size-federated	keycloak	META-INF/jpa-changelog-13.0.0.xml	2026-06-20 00:55:11.939584	94	EXECUTED	9:43c0c1055b6761b4b3e89de76d612ccf	modifyDataType columnName=CLIENT_ID, tableName=CLIENT_SCOPE_CLIENT; modifyDataType columnName=SCOPE_ID, tableName=CLIENT_SCOPE_CLIENT		\N	4.33.0	\N	\N	1916909404
13.0.0-KEYCLOAK-17992-recreate-constraints	keycloak	META-INF/jpa-changelog-13.0.0.xml	2026-06-20 00:55:11.940011	95	MARK_RAN	9:8bd711fd0330f4fe980494ca43ab1139	addNotNullConstraint columnName=CLIENT_ID, tableName=CLIENT_SCOPE_CLIENT; addNotNullConstraint columnName=SCOPE_ID, tableName=CLIENT_SCOPE_CLIENT; addPrimaryKey constraintName=C_CLI_SCOPE_BIND, tableName=CLIENT_SCOPE_CLIENT; createIndex indexName=...		\N	4.33.0	\N	\N	1916909404
json-string-accomodation-fixed	keycloak	META-INF/jpa-changelog-13.0.0.xml	2026-06-20 00:55:11.941561	96	EXECUTED	9:e07d2bc0970c348bb06fb63b1f82ddbf	addColumn tableName=REALM_ATTRIBUTE; update tableName=REALM_ATTRIBUTE; dropColumn columnName=VALUE, tableName=REALM_ATTRIBUTE; renameColumn newColumnName=VALUE, oldColumnName=VALUE_NEW, tableName=REALM_ATTRIBUTE		\N	4.33.0	\N	\N	1916909404
14.0.0-KEYCLOAK-11019	keycloak	META-INF/jpa-changelog-14.0.0.xml	2026-06-20 00:55:11.967397	97	EXECUTED	9:24fb8611e97f29989bea412aa38d12b7	createIndex indexName=IDX_OFFLINE_CSS_PRELOAD, tableName=OFFLINE_CLIENT_SESSION; createIndex indexName=IDX_OFFLINE_USS_BY_USER, tableName=OFFLINE_USER_SESSION; createIndex indexName=IDX_OFFLINE_USS_BY_USERSESS, tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
14.0.0-KEYCLOAK-18286	keycloak	META-INF/jpa-changelog-14.0.0.xml	2026-06-20 00:55:11.967866	98	MARK_RAN	9:259f89014ce2506ee84740cbf7163aa7	createIndex indexName=IDX_CLIENT_ATT_BY_NAME_VALUE, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
14.0.0-KEYCLOAK-18286-revert	keycloak	META-INF/jpa-changelog-14.0.0.xml	2026-06-20 00:55:11.972038	99	MARK_RAN	9:04baaf56c116ed19951cbc2cca584022	dropIndex indexName=IDX_CLIENT_ATT_BY_NAME_VALUE, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
14.0.0-KEYCLOAK-18286-supported-dbs	keycloak	META-INF/jpa-changelog-14.0.0.xml	2026-06-20 00:55:11.982091	100	EXECUTED	9:60ca84a0f8c94ec8c3504a5a3bc88ee8	createIndex indexName=IDX_CLIENT_ATT_BY_NAME_VALUE, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
14.0.0-KEYCLOAK-18286-unsupported-dbs	keycloak	META-INF/jpa-changelog-14.0.0.xml	2026-06-20 00:55:11.982532	101	MARK_RAN	9:d3d977031d431db16e2c181ce49d73e9	createIndex indexName=IDX_CLIENT_ATT_BY_NAME_VALUE, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
KEYCLOAK-17267-add-index-to-user-attributes	keycloak	META-INF/jpa-changelog-14.0.0.xml	2026-06-20 00:55:11.991126	102	EXECUTED	9:0b305d8d1277f3a89a0a53a659ad274c	createIndex indexName=IDX_USER_ATTRIBUTE_NAME, tableName=USER_ATTRIBUTE		\N	4.33.0	\N	\N	1916909404
KEYCLOAK-18146-add-saml-art-binding-identifier	keycloak	META-INF/jpa-changelog-14.0.0.xml	2026-06-20 00:55:11.992082	103	EXECUTED	9:2c374ad2cdfe20e2905a84c8fac48460	customChange		\N	4.33.0	\N	\N	1916909404
15.0.0-KEYCLOAK-18467	keycloak	META-INF/jpa-changelog-15.0.0.xml	2026-06-20 00:55:11.99339	104	EXECUTED	9:47a760639ac597360a8219f5b768b4de	addColumn tableName=REALM_LOCALIZATIONS; update tableName=REALM_LOCALIZATIONS; dropColumn columnName=TEXTS, tableName=REALM_LOCALIZATIONS; renameColumn newColumnName=TEXTS, oldColumnName=TEXTS_NEW, tableName=REALM_LOCALIZATIONS; addNotNullConstrai...		\N	4.33.0	\N	\N	1916909404
17.0.0-9562	keycloak	META-INF/jpa-changelog-17.0.0.xml	2026-06-20 00:55:12.002424	105	EXECUTED	9:a6272f0576727dd8cad2522335f5d99e	createIndex indexName=IDX_USER_SERVICE_ACCOUNT, tableName=USER_ENTITY		\N	4.33.0	\N	\N	1916909404
18.0.0-10625-IDX_ADMIN_EVENT_TIME	keycloak	META-INF/jpa-changelog-18.0.0.xml	2026-06-20 00:55:12.010981	106	EXECUTED	9:015479dbd691d9cc8669282f4828c41d	createIndex indexName=IDX_ADMIN_EVENT_TIME, tableName=ADMIN_EVENT_ENTITY		\N	4.33.0	\N	\N	1916909404
18.0.15-30992-index-consent	keycloak	META-INF/jpa-changelog-18.0.15.xml	2026-06-20 00:55:12.022717	107	EXECUTED	9:80071ede7a05604b1f4906f3bf3b00f0	createIndex indexName=IDX_USCONSENT_SCOPE_ID, tableName=USER_CONSENT_CLIENT_SCOPE		\N	4.33.0	\N	\N	1916909404
19.0.0-10135	keycloak	META-INF/jpa-changelog-19.0.0.xml	2026-06-20 00:55:12.023696	108	EXECUTED	9:9518e495fdd22f78ad6425cc30630221	customChange		\N	4.33.0	\N	\N	1916909404
20.0.0-12964-supported-dbs	keycloak	META-INF/jpa-changelog-20.0.0.xml	2026-06-20 00:55:12.032487	109	EXECUTED	9:e5f243877199fd96bcc842f27a1656ac	createIndex indexName=IDX_GROUP_ATT_BY_NAME_VALUE, tableName=GROUP_ATTRIBUTE		\N	4.33.0	\N	\N	1916909404
20.0.0-12964-supported-dbs-edb-migration	keycloak	META-INF/jpa-changelog-20.0.0.xml	2026-06-20 00:55:12.044447	110	EXECUTED	9:a6b18a8e38062df5793edbe064f4aecd	dropIndex indexName=IDX_GROUP_ATT_BY_NAME_VALUE, tableName=GROUP_ATTRIBUTE; createIndex indexName=IDX_GROUP_ATT_BY_NAME_VALUE, tableName=GROUP_ATTRIBUTE		\N	4.33.0	\N	\N	1916909404
20.0.0-12964-unsupported-dbs	keycloak	META-INF/jpa-changelog-20.0.0.xml	2026-06-20 00:55:12.044925	111	MARK_RAN	9:1a6fcaa85e20bdeae0a9ce49b41946a5	createIndex indexName=IDX_GROUP_ATT_BY_NAME_VALUE, tableName=GROUP_ATTRIBUTE		\N	4.33.0	\N	\N	1916909404
client-attributes-string-accomodation-fixed-pre-drop-index	keycloak	META-INF/jpa-changelog-20.0.0.xml	2026-06-20 00:55:12.045984	112	EXECUTED	9:04baaf56c116ed19951cbc2cca584022	dropIndex indexName=IDX_CLIENT_ATT_BY_NAME_VALUE, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
client-attributes-string-accomodation-fixed	keycloak	META-INF/jpa-changelog-20.0.0.xml	2026-06-20 00:55:12.047206	113	EXECUTED	9:3f332e13e90739ed0c35b0b25b7822ca	addColumn tableName=CLIENT_ATTRIBUTES; update tableName=CLIENT_ATTRIBUTES; dropColumn columnName=VALUE, tableName=CLIENT_ATTRIBUTES; renameColumn newColumnName=VALUE, oldColumnName=VALUE_NEW, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
client-attributes-string-accomodation-fixed-post-create-index	keycloak	META-INF/jpa-changelog-20.0.0.xml	2026-06-20 00:55:12.047532	114	MARK_RAN	9:bd2bd0fc7768cf0845ac96a8786fa735	createIndex indexName=IDX_CLIENT_ATT_BY_NAME_VALUE, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
21.0.2-17277	keycloak	META-INF/jpa-changelog-21.0.2.xml	2026-06-20 00:55:12.048574	115	EXECUTED	9:7ee1f7a3fb8f5588f171fb9a6ab623c0	customChange		\N	4.33.0	\N	\N	1916909404
21.1.0-19404	keycloak	META-INF/jpa-changelog-21.1.0.xml	2026-06-20 00:55:12.050846	116	EXECUTED	9:3d7e830b52f33676b9d64f7f2b2ea634	modifyDataType columnName=DECISION_STRATEGY, tableName=RESOURCE_SERVER_POLICY; modifyDataType columnName=LOGIC, tableName=RESOURCE_SERVER_POLICY; modifyDataType columnName=POLICY_ENFORCE_MODE, tableName=RESOURCE_SERVER		\N	4.33.0	\N	\N	1916909404
21.1.0-19404-2	keycloak	META-INF/jpa-changelog-21.1.0.xml	2026-06-20 00:55:12.051843	117	MARK_RAN	9:627d032e3ef2c06c0e1f73d2ae25c26c	addColumn tableName=RESOURCE_SERVER_POLICY; update tableName=RESOURCE_SERVER_POLICY; dropColumn columnName=DECISION_STRATEGY, tableName=RESOURCE_SERVER_POLICY; renameColumn newColumnName=DECISION_STRATEGY, oldColumnName=DECISION_STRATEGY_NEW, tabl...		\N	4.33.0	\N	\N	1916909404
22.0.0-17484-updated	keycloak	META-INF/jpa-changelog-22.0.0.xml	2026-06-20 00:55:12.05292	118	EXECUTED	9:90af0bfd30cafc17b9f4d6eccd92b8b3	customChange		\N	4.33.0	\N	\N	1916909404
23.0.0-12062	keycloak	META-INF/jpa-changelog-23.0.0.xml	2026-06-20 00:55:12.054392	120	EXECUTED	9:2168fbe728fec46ae9baf15bf80927b8	addColumn tableName=COMPONENT_CONFIG; update tableName=COMPONENT_CONFIG; dropColumn columnName=VALUE, tableName=COMPONENT_CONFIG; renameColumn newColumnName=VALUE, oldColumnName=VALUE_NEW, tableName=COMPONENT_CONFIG		\N	4.33.0	\N	\N	1916909404
23.0.0-17258	keycloak	META-INF/jpa-changelog-23.0.0.xml	2026-06-20 00:55:12.055292	121	EXECUTED	9:36506d679a83bbfda85a27ea1864dca8	addColumn tableName=EVENT_ENTITY		\N	4.33.0	\N	\N	1916909404
24.0.0-9758	keycloak	META-INF/jpa-changelog-24.0.0.xml	2026-06-20 00:55:12.091917	122	EXECUTED	9:502c557a5189f600f0f445a9b49ebbce	addColumn tableName=USER_ATTRIBUTE; addColumn tableName=FED_USER_ATTRIBUTE; createIndex indexName=USER_ATTR_LONG_VALUES, tableName=USER_ATTRIBUTE; createIndex indexName=FED_USER_ATTR_LONG_VALUES, tableName=FED_USER_ATTRIBUTE; createIndex indexName...		\N	4.33.0	\N	\N	1916909404
24.0.0-9758-2	keycloak	META-INF/jpa-changelog-24.0.0.xml	2026-06-20 00:55:12.092897	123	EXECUTED	9:bf0fdee10afdf597a987adbf291db7b2	customChange		\N	4.33.0	\N	\N	1916909404
24.0.0-26618-drop-index-if-present	keycloak	META-INF/jpa-changelog-24.0.0.xml	2026-06-20 00:55:12.09454	124	MARK_RAN	9:04baaf56c116ed19951cbc2cca584022	dropIndex indexName=IDX_CLIENT_ATT_BY_NAME_VALUE, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
24.0.0-26618-reindex	keycloak	META-INF/jpa-changelog-24.0.0.xml	2026-06-20 00:55:12.104964	125	EXECUTED	9:08707c0f0db1cef6b352db03a60edc7f	createIndex indexName=IDX_CLIENT_ATT_BY_NAME_VALUE, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
24.0.0-26618-edb-migration	keycloak	META-INF/jpa-changelog-24.0.0.xml	2026-06-20 00:55:12.115953	126	EXECUTED	9:2f684b29d414cd47efe3a3599f390741	dropIndex indexName=IDX_CLIENT_ATT_BY_NAME_VALUE, tableName=CLIENT_ATTRIBUTES; createIndex indexName=IDX_CLIENT_ATT_BY_NAME_VALUE, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
24.0.2-27228	keycloak	META-INF/jpa-changelog-24.0.2.xml	2026-06-20 00:55:12.116936	127	EXECUTED	9:eaee11f6b8aa25d2cc6a84fb86fc6238	customChange		\N	4.33.0	\N	\N	1916909404
24.0.2-27967-drop-index-if-present	keycloak	META-INF/jpa-changelog-24.0.2.xml	2026-06-20 00:55:12.117376	128	MARK_RAN	9:04baaf56c116ed19951cbc2cca584022	dropIndex indexName=IDX_CLIENT_ATT_BY_NAME_VALUE, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
24.0.2-27967-reindex	keycloak	META-INF/jpa-changelog-24.0.2.xml	2026-06-20 00:55:12.117894	129	MARK_RAN	9:d3d977031d431db16e2c181ce49d73e9	createIndex indexName=IDX_CLIENT_ATT_BY_NAME_VALUE, tableName=CLIENT_ATTRIBUTES		\N	4.33.0	\N	\N	1916909404
25.0.0-28265-tables	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.119133	130	EXECUTED	9:deda2df035df23388af95bbd36c17cef	addColumn tableName=OFFLINE_USER_SESSION; addColumn tableName=OFFLINE_CLIENT_SESSION		\N	4.33.0	\N	\N	1916909404
25.0.0-28265-index-creation	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.128522	131	EXECUTED	9:3e96709818458ae49f3c679ae58d263a	createIndex indexName=IDX_OFFLINE_USS_BY_LAST_SESSION_REFRESH, tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
25.0.0-28265-index-cleanup-uss-createdon	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.130377	132	EXECUTED	9:78ab4fc129ed5e8265dbcc3485fba92f	dropIndex indexName=IDX_OFFLINE_USS_CREATEDON, tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
25.0.0-28265-index-cleanup-uss-preload	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.132011	133	EXECUTED	9:de5f7c1f7e10994ed8b62e621d20eaab	dropIndex indexName=IDX_OFFLINE_USS_PRELOAD, tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
25.0.0-28265-index-cleanup-uss-by-usersess	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.134025	134	EXECUTED	9:6eee220d024e38e89c799417ec33667f	dropIndex indexName=IDX_OFFLINE_USS_BY_USERSESS, tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
25.0.0-28265-index-cleanup-css-preload	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.135847	135	EXECUTED	9:5411d2fb2891d3e8d63ddb55dfa3c0c9	dropIndex indexName=IDX_OFFLINE_CSS_PRELOAD, tableName=OFFLINE_CLIENT_SESSION		\N	4.33.0	\N	\N	1916909404
25.0.0-28265-index-2-mysql	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.136254	136	MARK_RAN	9:b7ef76036d3126bb83c2423bf4d449d6	createIndex indexName=IDX_OFFLINE_USS_BY_BROKER_SESSION_ID, tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
25.0.0-28265-index-2-not-mysql	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.14677	137	EXECUTED	9:23396cf51ab8bc1ae6f0cac7f9f6fcf7	createIndex indexName=IDX_OFFLINE_USS_BY_BROKER_SESSION_ID, tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
25.0.0-org	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.149413	138	EXECUTED	9:5c859965c2c9b9c72136c360649af157	createTable tableName=ORG; addUniqueConstraint constraintName=UK_ORG_NAME, tableName=ORG; addUniqueConstraint constraintName=UK_ORG_GROUP, tableName=ORG; createTable tableName=ORG_DOMAIN		\N	4.33.0	\N	\N	1916909404
unique-consentuser	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.151876	139	EXECUTED	9:5857626a2ea8767e9a6c66bf3a2cb32f	customChange; dropUniqueConstraint constraintName=UK_JKUWUVD56ONTGSUHOGM8UEWRT, tableName=USER_CONSENT; addUniqueConstraint constraintName=UK_LOCAL_CONSENT, tableName=USER_CONSENT; addUniqueConstraint constraintName=UK_EXTERNAL_CONSENT, tableName=...		\N	4.33.0	\N	\N	1916909404
unique-consentuser-edb-migration	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.153928	140	MARK_RAN	9:5857626a2ea8767e9a6c66bf3a2cb32f	customChange; dropUniqueConstraint constraintName=UK_JKUWUVD56ONTGSUHOGM8UEWRT, tableName=USER_CONSENT; addUniqueConstraint constraintName=UK_LOCAL_CONSENT, tableName=USER_CONSENT; addUniqueConstraint constraintName=UK_EXTERNAL_CONSENT, tableName=...		\N	4.33.0	\N	\N	1916909404
unique-consentuser-mysql	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.154525	141	MARK_RAN	9:b79478aad5adaa1bc428e31563f55e8e	customChange; dropUniqueConstraint constraintName=UK_JKUWUVD56ONTGSUHOGM8UEWRT, tableName=USER_CONSENT; addUniqueConstraint constraintName=UK_LOCAL_CONSENT, tableName=USER_CONSENT; addUniqueConstraint constraintName=UK_EXTERNAL_CONSENT, tableName=...		\N	4.33.0	\N	\N	1916909404
25.0.0-28861-index-creation	keycloak	META-INF/jpa-changelog-25.0.0.xml	2026-06-20 00:55:12.175343	142	EXECUTED	9:b9acb58ac958d9ada0fe12a5d4794ab1	createIndex indexName=IDX_PERM_TICKET_REQUESTER, tableName=RESOURCE_SERVER_PERM_TICKET; createIndex indexName=IDX_PERM_TICKET_OWNER, tableName=RESOURCE_SERVER_PERM_TICKET		\N	4.33.0	\N	\N	1916909404
26.0.0-org-alias	keycloak	META-INF/jpa-changelog-26.0.0.xml	2026-06-20 00:55:12.176978	143	EXECUTED	9:6ef7d63e4412b3c2d66ed179159886a4	addColumn tableName=ORG; update tableName=ORG; addNotNullConstraint columnName=ALIAS, tableName=ORG; addUniqueConstraint constraintName=UK_ORG_ALIAS, tableName=ORG		\N	4.33.0	\N	\N	1916909404
26.0.0-org-group	keycloak	META-INF/jpa-changelog-26.0.0.xml	2026-06-20 00:55:12.178807	144	EXECUTED	9:da8e8087d80ef2ace4f89d8c5b9ca223	addColumn tableName=KEYCLOAK_GROUP; update tableName=KEYCLOAK_GROUP; addNotNullConstraint columnName=TYPE, tableName=KEYCLOAK_GROUP; customChange		\N	4.33.0	\N	\N	1916909404
26.0.0-org-indexes	keycloak	META-INF/jpa-changelog-26.0.0.xml	2026-06-20 00:55:12.188719	145	EXECUTED	9:79b05dcd610a8c7f25ec05135eec0857	createIndex indexName=IDX_ORG_DOMAIN_ORG_ID, tableName=ORG_DOMAIN		\N	4.33.0	\N	\N	1916909404
26.0.0-org-group-membership	keycloak	META-INF/jpa-changelog-26.0.0.xml	2026-06-20 00:55:12.190215	146	EXECUTED	9:a6ace2ce583a421d89b01ba2a28dc2d4	addColumn tableName=USER_GROUP_MEMBERSHIP; update tableName=USER_GROUP_MEMBERSHIP; addNotNullConstraint columnName=MEMBERSHIP_TYPE, tableName=USER_GROUP_MEMBERSHIP		\N	4.33.0	\N	\N	1916909404
31296-persist-revoked-access-tokens	keycloak	META-INF/jpa-changelog-26.0.0.xml	2026-06-20 00:55:12.191287	147	EXECUTED	9:64ef94489d42a358e8304b0e245f0ed4	createTable tableName=REVOKED_TOKEN; addPrimaryKey constraintName=CONSTRAINT_RT, tableName=REVOKED_TOKEN		\N	4.33.0	\N	\N	1916909404
31725-index-persist-revoked-access-tokens	keycloak	META-INF/jpa-changelog-26.0.0.xml	2026-06-20 00:55:12.200273	148	EXECUTED	9:b994246ec2bf7c94da881e1d28782c7b	createIndex indexName=IDX_REV_TOKEN_ON_EXPIRE, tableName=REVOKED_TOKEN		\N	4.33.0	\N	\N	1916909404
26.0.0-idps-for-login	keycloak	META-INF/jpa-changelog-26.0.0.xml	2026-06-20 00:55:12.218143	149	EXECUTED	9:51f5fffadf986983d4bd59582c6c1604	addColumn tableName=IDENTITY_PROVIDER; createIndex indexName=IDX_IDP_REALM_ORG, tableName=IDENTITY_PROVIDER; createIndex indexName=IDX_IDP_FOR_LOGIN, tableName=IDENTITY_PROVIDER; customChange		\N	4.33.0	\N	\N	1916909404
26.0.0-32583-drop-redundant-index-on-client-session	keycloak	META-INF/jpa-changelog-26.0.0.xml	2026-06-20 00:55:12.219889	150	EXECUTED	9:24972d83bf27317a055d234187bb4af9	dropIndex indexName=IDX_US_SESS_ID_ON_CL_SESS, tableName=OFFLINE_CLIENT_SESSION		\N	4.33.0	\N	\N	1916909404
26.0.0.32582-remove-tables-user-session-user-session-note-and-client-session	keycloak	META-INF/jpa-changelog-26.0.0.xml	2026-06-20 00:55:12.222824	151	EXECUTED	9:febdc0f47f2ed241c59e60f58c3ceea5	dropTable tableName=CLIENT_SESSION_ROLE; dropTable tableName=CLIENT_SESSION_NOTE; dropTable tableName=CLIENT_SESSION_PROT_MAPPER; dropTable tableName=CLIENT_SESSION_AUTH_STATUS; dropTable tableName=CLIENT_USER_SESSION_NOTE; dropTable tableName=CLI...		\N	4.33.0	\N	\N	1916909404
26.0.0-33201-org-redirect-url	keycloak	META-INF/jpa-changelog-26.0.0.xml	2026-06-20 00:55:12.223515	152	EXECUTED	9:4d0e22b0ac68ebe9794fa9cb752ea660	addColumn tableName=ORG		\N	4.33.0	\N	\N	1916909404
29399-jdbc-ping-default	keycloak	META-INF/jpa-changelog-26.1.0.xml	2026-06-20 00:55:12.224962	153	EXECUTED	9:007dbe99d7203fca403b89d4edfdf21e	createTable tableName=JGROUPS_PING; addPrimaryKey constraintName=CONSTRAINT_JGROUPS_PING, tableName=JGROUPS_PING		\N	4.33.0	\N	\N	1916909404
26.1.0-34013	keycloak	META-INF/jpa-changelog-26.1.0.xml	2026-06-20 00:55:12.226164	154	EXECUTED	9:e6b686a15759aef99a6d758a5c4c6a26	addColumn tableName=ADMIN_EVENT_ENTITY		\N	4.33.0	\N	\N	1916909404
26.1.0-34380	keycloak	META-INF/jpa-changelog-26.1.0.xml	2026-06-20 00:55:12.226938	155	EXECUTED	9:ac8b9edb7c2b6c17a1c7a11fcf5ccf01	dropTable tableName=USERNAME_LOGIN_FAILURE		\N	4.33.0	\N	\N	1916909404
26.2.0-36750	keycloak	META-INF/jpa-changelog-26.2.0.xml	2026-06-20 00:55:12.227946	156	EXECUTED	9:b49ce951c22f7eb16480ff085640a33a	createTable tableName=SERVER_CONFIG		\N	4.33.0	\N	\N	1916909404
26.2.0-26106	keycloak	META-INF/jpa-changelog-26.2.0.xml	2026-06-20 00:55:12.228646	157	EXECUTED	9:b5877d5dab7d10ff3a9d209d7beb6680	addColumn tableName=CREDENTIAL		\N	4.33.0	\N	\N	1916909404
26.2.6-39866-duplicate	keycloak	META-INF/jpa-changelog-26.2.6.xml	2026-06-20 00:55:12.229934	158	EXECUTED	9:1dc67ccee24f30331db2cba4f372e40e	customChange		\N	4.33.0	\N	\N	1916909404
26.2.6-39866-uk	keycloak	META-INF/jpa-changelog-26.2.6.xml	2026-06-20 00:55:12.230694	159	EXECUTED	9:b70b76f47210cf0a5f4ef0e219eac7cd	addUniqueConstraint constraintName=UK_MIGRATION_VERSION, tableName=MIGRATION_MODEL		\N	4.33.0	\N	\N	1916909404
26.2.6-40088-duplicate	keycloak	META-INF/jpa-changelog-26.2.6.xml	2026-06-20 00:55:12.231525	160	EXECUTED	9:cc7e02ed69ab31979afb1982f9670e8f	customChange		\N	4.33.0	\N	\N	1916909404
26.2.6-40088-uk	keycloak	META-INF/jpa-changelog-26.2.6.xml	2026-06-20 00:55:12.232195	161	EXECUTED	9:5bb848128da7bc4595cc507383325241	addUniqueConstraint constraintName=UK_MIGRATION_UPDATE_TIME, tableName=MIGRATION_MODEL		\N	4.33.0	\N	\N	1916909404
26.3.0-groups-description	keycloak	META-INF/jpa-changelog-26.3.0.xml	2026-06-20 00:55:12.232946	162	EXECUTED	9:e1a3c05574326fb5b246b73b9a4c4d49	addColumn tableName=KEYCLOAK_GROUP		\N	4.33.0	\N	\N	1916909404
26.4.0-40933-saml-encryption-attributes	keycloak	META-INF/jpa-changelog-26.4.0.xml	2026-06-20 00:55:12.233775	163	EXECUTED	9:7e9eaba362ca105efdda202303a4fe49	customChange		\N	4.33.0	\N	\N	1916909404
26.4.0-51321	keycloak	META-INF/jpa-changelog-26.4.0.xml	2026-06-20 00:55:12.242041	164	EXECUTED	9:34bab2bc56f75ffd7e347c580874e306	createIndex indexName=IDX_EVENT_ENTITY_USER_ID_TYPE, tableName=EVENT_ENTITY		\N	4.33.0	\N	\N	1916909404
40343-workflow-state-table	keycloak	META-INF/jpa-changelog-26.4.0.xml	2026-06-20 00:55:12.258381	165	EXECUTED	9:ed3ab4723ceed210e5b5e60ac4562106	createTable tableName=WORKFLOW_STATE; addPrimaryKey constraintName=PK_WORKFLOW_STATE, tableName=WORKFLOW_STATE; addUniqueConstraint constraintName=UQ_WORKFLOW_RESOURCE, tableName=WORKFLOW_STATE; createIndex indexName=IDX_WORKFLOW_STATE_STEP, table...		\N	4.33.0	\N	\N	1916909404
26.5.0-index-offline-css-by-client	keycloak	META-INF/jpa-changelog-26.5.0.xml	2026-06-20 00:55:12.267932	166	EXECUTED	9:383e981ce95d16e32af757b7998820f7	createIndex indexName=IDX_OFFLINE_CSS_BY_CLIENT, tableName=OFFLINE_CLIENT_SESSION		\N	4.33.0	\N	\N	1916909404
26.5.0-index-offline-css-by-client-storage-provider	keycloak	META-INF/jpa-changelog-26.5.0.xml	2026-06-20 00:55:12.276919	167	EXECUTED	9:f5bc200e6fa7d7e483854dee535ca425	createIndex indexName=IDX_OFFLINE_CSS_BY_CLIENT_STORAGE_PROVIDER, tableName=OFFLINE_CLIENT_SESSION		\N	4.33.0	\N	\N	1916909404
26.5.0-idp-config-allow-null-fixed-drop-mssql-index	keycloak	META-INF/jpa-changelog-26.5.0.xml	2026-06-20 00:55:12.27724	168	MARK_RAN	9:50c51d2c98cd1d624eb1c485c3cf1f75	dropIndex indexName=IDX_IDP_FOR_LOGIN, tableName=IDENTITY_PROVIDER		\N	4.33.0	\N	\N	1916909404
26.5.0-idp-config-allow-null	keycloak	META-INF/jpa-changelog-26.5.0.xml	2026-06-20 00:55:12.279596	169	EXECUTED	9:b667fb087874303b324c1af7fae4f606	dropDefaultValue columnName=TRUST_EMAIL, tableName=IDENTITY_PROVIDER; dropNotNullConstraint columnName=TRUST_EMAIL, tableName=IDENTITY_PROVIDER; dropNotNullConstraint columnName=STORE_TOKEN, tableName=IDENTITY_PROVIDER; dropDefaultValue columnName...		\N	4.33.0	\N	\N	1916909404
26.5.0-idp-config-allow-null-fixed-create-mssql-index	keycloak	META-INF/jpa-changelog-26.5.0.xml	2026-06-20 00:55:12.279911	170	MARK_RAN	9:dcbbb24c151c3b0b59f12fede23cc94d	createIndex indexName=IDX_IDP_FOR_LOGIN, tableName=IDENTITY_PROVIDER		\N	4.33.0	\N	\N	1916909404
26.5.0-remove-workflow-provider-id-column	keycloak	META-INF/jpa-changelog-26.5.0.xml	2026-06-20 00:55:12.290183	171	EXECUTED	9:d8eeb324484d45e946d03b953e168b21	dropIndex indexName=IDX_WORKFLOW_STATE_PROVIDER, tableName=WORKFLOW_STATE; createIndex indexName=IDX_WORKFLOW_STATE_PROVIDER, tableName=WORKFLOW_STATE; dropColumn columnName=WORKFLOW_PROVIDER_ID, tableName=WORKFLOW_STATE		\N	4.33.0	\N	\N	1916909404
26.5.0-add-remember-me	keycloak	META-INF/jpa-changelog-26.5.0.xml	2026-06-20 00:55:12.291311	172	EXECUTED	9:a7273ea8b21bd2f674c9c49141999f05	addColumn tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
26.5.0-add-sess-refresh-idx	keycloak	META-INF/jpa-changelog-26.5.0.xml	2026-06-20 00:55:12.301763	173	EXECUTED	9:ce49383d317ccbcd3434d1f21172b0b7	createIndex indexName=IDX_USER_SESSION_EXPIRATION_CREATED, tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
26.5.0-add-sess-create-idx	keycloak	META-INF/jpa-changelog-26.5.0.xml	2026-06-20 00:55:12.311915	174	EXECUTED	9:aaee09e23a4d8468fbc5c51b7b314c58	createIndex indexName=IDX_USER_SESSION_EXPIRATION_LAST_REFRESH, tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
26.5.0-drop-sess-refresh-idx	keycloak	META-INF/jpa-changelog-26.5.0.xml	2026-06-20 00:55:12.313986	175	EXECUTED	9:f0082210b6ccbbaf81287c27aa23753c	dropIndex indexName=IDX_OFFLINE_USS_BY_LAST_SESSION_REFRESH, tableName=OFFLINE_USER_SESSION		\N	4.33.0	\N	\N	1916909404
26.5.0-mysql-mariadb-default-charset-collation	keycloak	META-INF/jpa-changelog-26.5.0.xml	2026-06-20 00:55:12.314342	176	MARK_RAN	9:1b383fa60d2db0a8952b365e725f9d16	customChange		\N	4.33.0	\N	\N	1916909404
26.5.0-invitations-table-fixed2	keycloak	META-INF/jpa-changelog-26.5.0.xml	2026-06-20 00:55:12.342678	177	EXECUTED	9:322cb11fc03181903dcd67a54f8b3cf0	createTable tableName=ORG_INVITATION; addForeignKeyConstraint baseTableName=ORG_INVITATION, constraintName=FK_ORG_INVITATION_ORG, referencedTableName=ORG; createIndex indexName=IDX_ORG_INVITATION_ORG_ID, tableName=ORG_INVITATION; createIndex index...		\N	4.33.0	\N	\N	1916909404
26.6.0-45009-broker-link-user-id	keycloak	META-INF/jpa-changelog-26.6.0.xml	2026-06-20 00:55:12.352874	178	EXECUTED	9:05026bbbc8d2ead5afcbda2f5fdf3a2b	createIndex indexName=IDX_BROKER_LINK_USER_ID, tableName=BROKER_LINK		\N	4.33.0	\N	\N	1916909404
26.6.0-45009-broker-link-identity-provider	keycloak	META-INF/jpa-changelog-26.6.0.xml	2026-06-20 00:55:12.363087	179	EXECUTED	9:7d9a0253c9de7be754efef8bba4265bd	createIndex indexName=IDX_BROKER_LINK_IDENTITY_PROVIDER, tableName=BROKER_LINK		\N	4.33.0	\N	\N	1916909404
26.6.0-org-group-relationship	keycloak	META-INF/jpa-changelog-26.6.0.xml	2026-06-20 00:55:12.373422	180	EXECUTED	9:05685853fba030f53548ac6bf23245e3	addColumn tableName=KEYCLOAK_GROUP; addForeignKeyConstraint baseTableName=KEYCLOAK_GROUP, constraintName=FK_GROUP_ORGANIZATION, referencedTableName=ORG; createIndex indexName=IDX_GROUP_ORG_ID, tableName=KEYCLOAK_GROUP		\N	4.33.0	\N	\N	1916909404
26.6.0-44424-index-css-user-session-and-offline	keycloak	META-INF/jpa-changelog-26.6.0.xml	2026-06-20 00:55:12.384023	181	EXECUTED	9:a704d8598df241a3fd3cb91b6ab4b2d4	createIndex indexName=IDX_OFFLINE_CSS_BY_USER_SESSION_AND_OFFLINE, tableName=OFFLINE_CLIENT_SESSION		\N	4.33.0	\N	\N	1916909404
26.6.0-44424-create-realm-in-client-session	keycloak	META-INF/jpa-changelog-26.6.0.xml	2026-06-20 00:55:12.385646	182	EXECUTED	9:77dbbc72d943e98cfe472ba8cc56a31c	addColumn tableName=OFFLINE_CLIENT_SESSION		\N	4.33.0	\N	\N	1916909404
26.6.0-44424-set-realm-in-client-session	keycloak	META-INF/jpa-changelog-26.6.0.xml	2026-06-20 00:55:12.386807	183	EXECUTED	9:3964a3148d32a55ef81126e23cdf6721	customChange		\N	4.33.0	\N	\N	1916909404
26.6.0-44424-idx-css-realm-and-clients	keycloak	META-INF/jpa-changelog-26.6.0.xml	2026-06-20 00:55:12.396987	184	EXECUTED	9:a093877fff41185ac24103be80e00968	createIndex indexName=IDX_OFFLINE_CSS_BY_CLIENT_AND_REALM, tableName=OFFLINE_CLIENT_SESSION		\N	4.33.0	\N	\N	1916909404
26.6.0-add-last-modified-timestamp-user	keycloak	META-INF/jpa-changelog-26.6.0.xml	2026-06-20 00:55:12.398194	185	EXECUTED	9:8aa583d2cdd9e913dff42fecd626c560	addColumn tableName=USER_ENTITY		\N	4.33.0	\N	\N	1916909404
26.6.0-add-timestamps-group	keycloak	META-INF/jpa-changelog-26.6.0.xml	2026-06-20 00:55:12.399246	186	EXECUTED	9:4363d45dc25105a3fc5db9ff6936b0a9	addColumn tableName=KEYCLOAK_GROUP		\N	4.33.0	\N	\N	1916909404
26.6.0-43829-user-created-timestamp-index	keycloak	META-INF/jpa-changelog-26.6.0.xml	2026-06-20 00:55:12.408519	187	EXECUTED	9:f2531a49b8bb21a7a97966d88fd1a411	createIndex indexName=IDX_USER_CREATED_TIMESTAMP, tableName=USER_ENTITY		\N	4.33.0	\N	\N	1916909404
26.6.0-48716-create-mssql-idp-index	keycloak	META-INF/jpa-changelog-26.6.0.xml	2026-06-20 00:55:12.408873	188	MARK_RAN	9:dcbbb24c151c3b0b59f12fede23cc94d	createIndex indexName=IDX_IDP_FOR_LOGIN, tableName=IDENTITY_PROVIDER		\N	4.33.0	\N	\N	1916909404
\.


--
-- Data for Name: databasechangeloglock; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.databasechangeloglock (id, locked, lockgranted, lockedby) FROM stdin;
1	f	\N	\N
1000	f	\N	\N
\.


--
-- Data for Name: default_client_scope; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.default_client_scope (realm_id, scope_id, default_scope) FROM stdin;
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	b511e099-6962-4dda-bdef-b9f25ce57db8	f
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	48d8e6fc-1e74-4bcc-bd96-8e90f7ad329d	t
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	251977c1-46b1-4844-bb5a-425e2212ef0e	t
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	3a505778-d76c-4957-af68-9019361f5fc9	t
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	a9d7c3c0-0351-4d08-aa53-eeed1f01e858	t
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	6b85c7f3-6922-4d08-94a1-38276d9cc804	f
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	8fc9262c-e47a-47e5-879b-4e1df207ea2f	f
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	dff93946-cd22-41b2-867d-350628a3e044	t
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	16b30cc6-85ff-4ba9-bb1f-85f802271826	t
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd	f
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	3ae652cf-ae49-4be9-9c9d-70ceef556823	t
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	eff33a74-12c5-460d-94dc-5ef255d0a1c2	t
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	a7467a26-8787-424e-8615-765968eb03a2	f
\.


--
-- Data for Name: event_entity; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.event_entity (id, client_id, details_json, error, ip_address, realm_id, session_id, event_time, type, user_id, details_json_long_value) FROM stdin;
\.


--
-- Data for Name: fed_user_attribute; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.fed_user_attribute (id, name, user_id, realm_id, storage_provider_id, value, long_value_hash, long_value_hash_lower_case, long_value) FROM stdin;
\.


--
-- Data for Name: fed_user_consent; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.fed_user_consent (id, client_id, user_id, realm_id, storage_provider_id, created_date, last_updated_date, client_storage_provider, external_client_id) FROM stdin;
\.


--
-- Data for Name: fed_user_consent_cl_scope; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.fed_user_consent_cl_scope (user_consent_id, scope_id) FROM stdin;
\.


--
-- Data for Name: fed_user_credential; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.fed_user_credential (id, salt, type, created_date, user_id, realm_id, storage_provider_id, user_label, secret_data, credential_data, priority) FROM stdin;
\.


--
-- Data for Name: fed_user_group_membership; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.fed_user_group_membership (group_id, user_id, realm_id, storage_provider_id) FROM stdin;
\.


--
-- Data for Name: fed_user_required_action; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.fed_user_required_action (required_action, user_id, realm_id, storage_provider_id) FROM stdin;
\.


--
-- Data for Name: fed_user_role_mapping; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.fed_user_role_mapping (role_id, user_id, realm_id, storage_provider_id) FROM stdin;
\.


--
-- Data for Name: federated_identity; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.federated_identity (identity_provider, realm_id, federated_user_id, federated_username, token, user_id) FROM stdin;
\.


--
-- Data for Name: federated_user; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.federated_user (id, storage_provider_id, realm_id) FROM stdin;
\.


--
-- Data for Name: group_attribute; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.group_attribute (id, name, value, group_id) FROM stdin;
\.


--
-- Data for Name: group_role_mapping; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.group_role_mapping (role_id, group_id) FROM stdin;
\.


--
-- Data for Name: identity_provider; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.identity_provider (internal_id, enabled, provider_alias, provider_id, store_token, authenticate_by_default, realm_id, add_token_role, trust_email, first_broker_login_flow_id, post_broker_login_flow_id, provider_display_name, link_only, organization_id, hide_on_login) FROM stdin;
\.


--
-- Data for Name: identity_provider_config; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.identity_provider_config (identity_provider_id, value, name) FROM stdin;
\.


--
-- Data for Name: identity_provider_mapper; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.identity_provider_mapper (id, name, idp_alias, idp_mapper_name, realm_id) FROM stdin;
\.


--
-- Data for Name: idp_mapper_config; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.idp_mapper_config (idp_mapper_id, value, name) FROM stdin;
\.


--
-- Data for Name: jgroups_ping; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.jgroups_ping (address, name, cluster_name, ip, coord) FROM stdin;
\.


--
-- Data for Name: keycloak_group; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.keycloak_group (id, name, parent_group, realm_id, type, description, org_id, created_timestamp, last_modified_timestamp) FROM stdin;
\.


--
-- Data for Name: keycloak_role; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.keycloak_role (id, client_realm_constraint, client_role, description, name, realm_id, client, realm) FROM stdin;
5f661961-94d9-49fb-9627-50430fff0145	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	f	${role_default-roles}	default-roles-master	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	\N	\N
5c4f9699-e858-4348-8c6e-7ed861b76b81	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	f	${role_create-realm}	create-realm	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	\N	\N
d74b8256-c52a-45ef-86e8-b8ec3391ee89	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	f	${role_admin}	admin	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	\N	\N
ac1db954-21a4-48de-a7fe-e89bc65ebb56	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_create-client}	create-client	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
d8e294ad-eff4-43ab-8031-5839928f0e19	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_view-realm}	view-realm	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
f78242be-3ae2-4d42-a72e-d287fffad8a7	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_view-users}	view-users	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
f3ab0b2a-359f-439f-8453-5c76ed3f1676	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_view-clients}	view-clients	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
de03e282-b1a8-40e8-bd0a-f058a4f1e97c	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_view-events}	view-events	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
615bd060-c696-424e-9a22-735dab494777	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_view-identity-providers}	view-identity-providers	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
3f2be762-9a42-4d49-821c-337339e8e8bc	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_view-authorization}	view-authorization	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
d5af39bc-78da-4524-8c9d-8d7bbbf9a82f	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_manage-realm}	manage-realm	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
196f1e41-4795-427f-8cdd-422c91be41f9	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_manage-users}	manage-users	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
71fe12d4-2f50-40f6-baeb-422ed6227bc0	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_manage-clients}	manage-clients	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
40069e55-8fb4-428c-86e2-46e097d56085	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_manage-events}	manage-events	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
adc595c6-3459-4e54-8640-3a32462b46e7	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_manage-identity-providers}	manage-identity-providers	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
6e81b2a0-eddc-4532-b510-e7918b84b274	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_manage-authorization}	manage-authorization	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
3bc6395e-80b8-4222-a37f-cc037e766a64	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_query-users}	query-users	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
17efd815-faca-4f67-9c4b-0f85017ce5e8	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_query-clients}	query-clients	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
e0d52462-d5da-4d44-a134-8b6b7cbfb3be	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_query-realms}	query-realms	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
5980979e-e8e4-4d2d-a302-2273157922b0	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_query-groups}	query-groups	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
c6fb3296-8dfe-4257-ab78-976e8fa51c33	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	t	${role_view-profile}	view-profile	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	\N
681c4058-b23f-4a15-b885-2ac090faa232	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	t	${role_manage-account}	manage-account	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	\N
8a8a7751-24ed-486d-84ea-a7ff627d6648	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	t	${role_manage-account-links}	manage-account-links	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	\N
d3a26b72-57d8-4875-8563-a4641a526415	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	t	${role_view-applications}	view-applications	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	\N
e52928e0-7dc9-4e81-b9d1-aeecd3b0aa92	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	t	${role_view-consent}	view-consent	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	\N
accd35fd-d802-4188-99a1-3040f7987b96	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	t	${role_manage-consent}	manage-consent	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	\N
37b1c2a2-23c5-498c-b596-eb4550ad62fc	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	t	${role_view-groups}	view-groups	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	\N
9ee7c2e1-6a36-4f74-9181-7fb2c26adf73	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	t	${role_delete-account}	delete-account	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	\N
986841d9-43c2-4dcd-899d-d677e69f821e	46baaef2-287a-4ef6-bbe2-39491e649624	t	${role_read-token}	read-token	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	46baaef2-287a-4ef6-bbe2-39491e649624	\N
b62a30b9-2f46-4b64-8552-8699c982eed0	9e64fa07-4221-4426-b056-6d4e83cc7021	t	${role_impersonation}	impersonation	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	9e64fa07-4221-4426-b056-6d4e83cc7021	\N
10fbb334-c415-430e-9b09-496038b00760	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	f	${role_offline-access}	offline_access	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	\N	\N
97095622-d040-4d6a-b137-24db9fe89025	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	f	${role_uma_authorization}	uma_authorization	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	\N	\N
\.


--
-- Data for Name: migration_model; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.migration_model (id, version, update_time) FROM stdin;
w8rkn	26.6.3	1781916915
\.


--
-- Data for Name: offline_client_session; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.offline_client_session (user_session_id, client_id, offline_flag, "timestamp", data, client_storage_provider, external_client_id, version, realm_id) FROM stdin;
GVzZDxudHhPGebLUi7gvQlRr	89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	0	1781917285	{"authMethod":"openid-connect","redirectUri":"http://localhost:8080/admin/master/console/#/master/users/f35a80a3-4d9f-4a68-997e-c470f30f2281/role-mapping","notes":{"clientId":"89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1","iss":"http://localhost:8080/realms/master","startedAt":"1781917042","response_type":"code","level-of-authentication":"-1","code_challenge_method":"S256","nonce":"e8544eaa-772e-4c18-a84e-260bf4972715","response_mode":"query","scope":"openid","userSessionStartedAt":"1781917042","redirect_uri":"http://localhost:8080/admin/master/console/#/master/users/f35a80a3-4d9f-4a68-997e-c470f30f2281/role-mapping","state":"2f92909a-be2d-412e-b98c-d2bad98d1279","code_challenge":"gkoOX5O5PBTLEZY1zFQByY5BoNNVtsGZKGSYBhNYWg8"}}	local	local	2	eb86d28d-cdbb-4987-839a-7f8ed73f98ee
\.


--
-- Data for Name: offline_user_session; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.offline_user_session (user_session_id, user_id, realm_id, created_on, offline_flag, data, last_session_refresh, broker_session_id, version, remember_me) FROM stdin;
GVzZDxudHhPGebLUi7gvQlRr	f35a80a3-4d9f-4a68-997e-c470f30f2281	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	1781917042	0	{"ipAddress":"192.168.65.1","authMethod":"openid-connect","rememberMe":false,"started":0,"notes":{"KC_DEVICE_NOTE":"eyJpcEFkZHJlc3MiOiIxOTIuMTY4LjY1LjEiLCJvcyI6Ik1hYyBPUyBYIiwib3NWZXJzaW9uIjoiMTAuMTUuNyIsImJyb3dzZXIiOiJDaHJvbWUvMTQ5LjAuMCIsImRldmljZSI6Ik1hYyIsImxhc3RBY2Nlc3MiOjAsIm1vYmlsZSI6ZmFsc2V9","AUTH_TIME":"1781917042","authenticators-completed":"{\\"71654bfb-3e83-4a08-89f6-be2f775e4929\\":1781917042}"},"state":"LOGGED_IN"}	1781917285	\N	2	f
\.


--
-- Data for Name: org; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.org (id, enabled, realm_id, group_id, name, description, alias, redirect_url) FROM stdin;
\.


--
-- Data for Name: org_domain; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.org_domain (id, name, verified, org_id) FROM stdin;
\.


--
-- Data for Name: org_invitation; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.org_invitation (id, organization_id, email, first_name, last_name, created_at, expires_at, invite_link) FROM stdin;
\.


--
-- Data for Name: policy_config; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.policy_config (policy_id, name, value) FROM stdin;
\.


--
-- Data for Name: protocol_mapper; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.protocol_mapper (id, name, protocol, protocol_mapper_name, client_id, client_scope_id) FROM stdin;
ae6abd56-5c69-4e25-96ef-e5635b6eb4a3	audience resolve	openid-connect	oidc-audience-resolve-mapper	e16e1bd2-ccc5-418d-87c3-fc45f25526b7	\N
554da0b8-0d53-4a9a-b9bc-2d1ddb5ad58f	locale	openid-connect	oidc-usermodel-attribute-mapper	89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	\N
ae4f654f-e913-4578-aa8c-5232c270d746	role list	saml	saml-role-list-mapper	\N	48d8e6fc-1e74-4bcc-bd96-8e90f7ad329d
e993d32d-63d5-4f12-9fdd-3082173451f6	organization	saml	saml-organization-membership-mapper	\N	251977c1-46b1-4844-bb5a-425e2212ef0e
d6642017-5abb-4eb3-aada-c7f2769a0a0b	full name	openid-connect	oidc-full-name-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
b47b4346-da02-4c05-aa65-7157d9644ec0	family name	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
2123d8bc-51a8-42bb-b87f-6416f5af8c45	given name	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
421eff92-5456-49d1-868c-37875f3fc2bd	middle name	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
04d06bff-b4b8-4b40-b904-0795343d9942	nickname	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
817d4a4e-29a7-4ead-a833-5ceba6bdb1a4	username	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
584a7438-18e6-409e-93b9-30a98bad3d9d	profile	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
fbd992e5-6526-4b9b-a9bb-1dd62c62c4ac	picture	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
9e7852e6-735d-4539-98df-1615865e4aa4	website	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
dd0ad071-2166-4380-8d92-4d3e8e7b648d	gender	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
89c461f7-cdf6-48c6-8949-f68aa5130d23	birthdate	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
880b452e-7655-4037-bf00-7c62d1129b0c	zoneinfo	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
5a46872b-050b-4a90-b4ff-681c7ea1078c	locale	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
378ff88d-aae2-4f13-bfda-fdba932d0bf3	updated at	openid-connect	oidc-usermodel-attribute-mapper	\N	3a505778-d76c-4957-af68-9019361f5fc9
06901a48-3ef5-437b-988a-9b886b42bd67	email	openid-connect	oidc-usermodel-attribute-mapper	\N	a9d7c3c0-0351-4d08-aa53-eeed1f01e858
8a5e8b29-a29e-46b5-90a6-94633cfd464a	email verified	openid-connect	oidc-usermodel-property-mapper	\N	a9d7c3c0-0351-4d08-aa53-eeed1f01e858
534f87c2-6bd2-4f64-af4c-79cbe5e3f9b8	address	openid-connect	oidc-address-mapper	\N	6b85c7f3-6922-4d08-94a1-38276d9cc804
543b1b4f-b0af-47e0-b775-404f2d7b63aa	phone number	openid-connect	oidc-usermodel-attribute-mapper	\N	8fc9262c-e47a-47e5-879b-4e1df207ea2f
78c4f2da-e724-4835-82e3-174ee16f3767	phone number verified	openid-connect	oidc-usermodel-attribute-mapper	\N	8fc9262c-e47a-47e5-879b-4e1df207ea2f
8f61003b-74d9-4cbc-b8bf-ef5b06cae8ef	realm roles	openid-connect	oidc-usermodel-realm-role-mapper	\N	dff93946-cd22-41b2-867d-350628a3e044
5393b5b2-db3d-49db-b24e-83aa5bbcca43	client roles	openid-connect	oidc-usermodel-client-role-mapper	\N	dff93946-cd22-41b2-867d-350628a3e044
2ef85bd8-2808-44c4-a804-2561cd6c5d34	audience resolve	openid-connect	oidc-audience-resolve-mapper	\N	dff93946-cd22-41b2-867d-350628a3e044
f64db3b6-a89f-4c9f-87b2-d96405a27181	allowed web origins	openid-connect	oidc-allowed-origins-mapper	\N	16b30cc6-85ff-4ba9-bb1f-85f802271826
633b8d3b-9208-4052-b1c7-aabe705f9479	upn	openid-connect	oidc-usermodel-attribute-mapper	\N	6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd
ff0206da-8501-4078-9646-ede3f2304f60	groups	openid-connect	oidc-usermodel-realm-role-mapper	\N	6356fa7c-dbb4-4cdf-a290-4a45dff7e7dd
43c2d1a7-c647-43c3-9457-e3107e634f02	acr loa level	openid-connect	oidc-acr-mapper	\N	3ae652cf-ae49-4be9-9c9d-70ceef556823
134421d6-1290-496b-ad09-b0a887b806da	auth_time	openid-connect	oidc-usersessionmodel-note-mapper	\N	eff33a74-12c5-460d-94dc-5ef255d0a1c2
144c8b70-d272-4c4c-bf3f-10ff7edfa8f6	sub	openid-connect	oidc-sub-mapper	\N	eff33a74-12c5-460d-94dc-5ef255d0a1c2
6ca113d3-dd50-4749-8604-1291e659025f	Client ID	openid-connect	oidc-usersessionmodel-note-mapper	\N	37d2fdba-cb42-41ca-b47e-3439a724dcdf
1a10382f-1460-4665-858f-4dd49bdc8077	Client Host	openid-connect	oidc-usersessionmodel-note-mapper	\N	37d2fdba-cb42-41ca-b47e-3439a724dcdf
3d61dff0-76fa-4e86-b922-903fe4d62ff7	Client IP Address	openid-connect	oidc-usersessionmodel-note-mapper	\N	37d2fdba-cb42-41ca-b47e-3439a724dcdf
7b679bcb-8356-42c8-b058-b778952f5ead	organization	openid-connect	oidc-organization-membership-mapper	\N	a7467a26-8787-424e-8615-765968eb03a2
\.


--
-- Data for Name: protocol_mapper_config; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.protocol_mapper_config (protocol_mapper_id, value, name) FROM stdin;
554da0b8-0d53-4a9a-b9bc-2d1ddb5ad58f	true	introspection.token.claim
554da0b8-0d53-4a9a-b9bc-2d1ddb5ad58f	true	userinfo.token.claim
554da0b8-0d53-4a9a-b9bc-2d1ddb5ad58f	locale	user.attribute
554da0b8-0d53-4a9a-b9bc-2d1ddb5ad58f	true	id.token.claim
554da0b8-0d53-4a9a-b9bc-2d1ddb5ad58f	true	access.token.claim
554da0b8-0d53-4a9a-b9bc-2d1ddb5ad58f	locale	claim.name
554da0b8-0d53-4a9a-b9bc-2d1ddb5ad58f	String	jsonType.label
ae4f654f-e913-4578-aa8c-5232c270d746	false	single
ae4f654f-e913-4578-aa8c-5232c270d746	Basic	attribute.nameformat
ae4f654f-e913-4578-aa8c-5232c270d746	Role	attribute.name
04d06bff-b4b8-4b40-b904-0795343d9942	true	introspection.token.claim
04d06bff-b4b8-4b40-b904-0795343d9942	true	userinfo.token.claim
04d06bff-b4b8-4b40-b904-0795343d9942	nickname	user.attribute
04d06bff-b4b8-4b40-b904-0795343d9942	true	id.token.claim
04d06bff-b4b8-4b40-b904-0795343d9942	true	access.token.claim
04d06bff-b4b8-4b40-b904-0795343d9942	nickname	claim.name
04d06bff-b4b8-4b40-b904-0795343d9942	String	jsonType.label
2123d8bc-51a8-42bb-b87f-6416f5af8c45	true	introspection.token.claim
2123d8bc-51a8-42bb-b87f-6416f5af8c45	true	userinfo.token.claim
2123d8bc-51a8-42bb-b87f-6416f5af8c45	firstName	user.attribute
2123d8bc-51a8-42bb-b87f-6416f5af8c45	true	id.token.claim
2123d8bc-51a8-42bb-b87f-6416f5af8c45	true	access.token.claim
2123d8bc-51a8-42bb-b87f-6416f5af8c45	given_name	claim.name
2123d8bc-51a8-42bb-b87f-6416f5af8c45	String	jsonType.label
378ff88d-aae2-4f13-bfda-fdba932d0bf3	true	introspection.token.claim
378ff88d-aae2-4f13-bfda-fdba932d0bf3	true	userinfo.token.claim
378ff88d-aae2-4f13-bfda-fdba932d0bf3	updatedAt	user.attribute
378ff88d-aae2-4f13-bfda-fdba932d0bf3	true	id.token.claim
378ff88d-aae2-4f13-bfda-fdba932d0bf3	true	access.token.claim
378ff88d-aae2-4f13-bfda-fdba932d0bf3	updated_at	claim.name
378ff88d-aae2-4f13-bfda-fdba932d0bf3	long	jsonType.label
421eff92-5456-49d1-868c-37875f3fc2bd	true	introspection.token.claim
421eff92-5456-49d1-868c-37875f3fc2bd	true	userinfo.token.claim
421eff92-5456-49d1-868c-37875f3fc2bd	middleName	user.attribute
421eff92-5456-49d1-868c-37875f3fc2bd	true	id.token.claim
421eff92-5456-49d1-868c-37875f3fc2bd	true	access.token.claim
421eff92-5456-49d1-868c-37875f3fc2bd	middle_name	claim.name
421eff92-5456-49d1-868c-37875f3fc2bd	String	jsonType.label
584a7438-18e6-409e-93b9-30a98bad3d9d	true	introspection.token.claim
584a7438-18e6-409e-93b9-30a98bad3d9d	true	userinfo.token.claim
584a7438-18e6-409e-93b9-30a98bad3d9d	profile	user.attribute
584a7438-18e6-409e-93b9-30a98bad3d9d	true	id.token.claim
584a7438-18e6-409e-93b9-30a98bad3d9d	true	access.token.claim
584a7438-18e6-409e-93b9-30a98bad3d9d	profile	claim.name
584a7438-18e6-409e-93b9-30a98bad3d9d	String	jsonType.label
5a46872b-050b-4a90-b4ff-681c7ea1078c	true	introspection.token.claim
5a46872b-050b-4a90-b4ff-681c7ea1078c	true	userinfo.token.claim
5a46872b-050b-4a90-b4ff-681c7ea1078c	locale	user.attribute
5a46872b-050b-4a90-b4ff-681c7ea1078c	true	id.token.claim
5a46872b-050b-4a90-b4ff-681c7ea1078c	true	access.token.claim
5a46872b-050b-4a90-b4ff-681c7ea1078c	locale	claim.name
5a46872b-050b-4a90-b4ff-681c7ea1078c	String	jsonType.label
817d4a4e-29a7-4ead-a833-5ceba6bdb1a4	true	introspection.token.claim
817d4a4e-29a7-4ead-a833-5ceba6bdb1a4	true	userinfo.token.claim
817d4a4e-29a7-4ead-a833-5ceba6bdb1a4	username	user.attribute
817d4a4e-29a7-4ead-a833-5ceba6bdb1a4	true	id.token.claim
817d4a4e-29a7-4ead-a833-5ceba6bdb1a4	true	access.token.claim
817d4a4e-29a7-4ead-a833-5ceba6bdb1a4	preferred_username	claim.name
817d4a4e-29a7-4ead-a833-5ceba6bdb1a4	String	jsonType.label
880b452e-7655-4037-bf00-7c62d1129b0c	true	introspection.token.claim
880b452e-7655-4037-bf00-7c62d1129b0c	true	userinfo.token.claim
880b452e-7655-4037-bf00-7c62d1129b0c	zoneinfo	user.attribute
880b452e-7655-4037-bf00-7c62d1129b0c	true	id.token.claim
880b452e-7655-4037-bf00-7c62d1129b0c	true	access.token.claim
880b452e-7655-4037-bf00-7c62d1129b0c	zoneinfo	claim.name
880b452e-7655-4037-bf00-7c62d1129b0c	String	jsonType.label
89c461f7-cdf6-48c6-8949-f68aa5130d23	true	introspection.token.claim
89c461f7-cdf6-48c6-8949-f68aa5130d23	true	userinfo.token.claim
89c461f7-cdf6-48c6-8949-f68aa5130d23	birthdate	user.attribute
89c461f7-cdf6-48c6-8949-f68aa5130d23	true	id.token.claim
89c461f7-cdf6-48c6-8949-f68aa5130d23	true	access.token.claim
89c461f7-cdf6-48c6-8949-f68aa5130d23	birthdate	claim.name
89c461f7-cdf6-48c6-8949-f68aa5130d23	String	jsonType.label
9e7852e6-735d-4539-98df-1615865e4aa4	true	introspection.token.claim
9e7852e6-735d-4539-98df-1615865e4aa4	true	userinfo.token.claim
9e7852e6-735d-4539-98df-1615865e4aa4	website	user.attribute
9e7852e6-735d-4539-98df-1615865e4aa4	true	id.token.claim
9e7852e6-735d-4539-98df-1615865e4aa4	true	access.token.claim
9e7852e6-735d-4539-98df-1615865e4aa4	website	claim.name
9e7852e6-735d-4539-98df-1615865e4aa4	String	jsonType.label
b47b4346-da02-4c05-aa65-7157d9644ec0	true	introspection.token.claim
b47b4346-da02-4c05-aa65-7157d9644ec0	true	userinfo.token.claim
b47b4346-da02-4c05-aa65-7157d9644ec0	lastName	user.attribute
b47b4346-da02-4c05-aa65-7157d9644ec0	true	id.token.claim
b47b4346-da02-4c05-aa65-7157d9644ec0	true	access.token.claim
b47b4346-da02-4c05-aa65-7157d9644ec0	family_name	claim.name
b47b4346-da02-4c05-aa65-7157d9644ec0	String	jsonType.label
d6642017-5abb-4eb3-aada-c7f2769a0a0b	true	introspection.token.claim
d6642017-5abb-4eb3-aada-c7f2769a0a0b	true	userinfo.token.claim
d6642017-5abb-4eb3-aada-c7f2769a0a0b	true	id.token.claim
d6642017-5abb-4eb3-aada-c7f2769a0a0b	true	access.token.claim
dd0ad071-2166-4380-8d92-4d3e8e7b648d	true	introspection.token.claim
dd0ad071-2166-4380-8d92-4d3e8e7b648d	true	userinfo.token.claim
dd0ad071-2166-4380-8d92-4d3e8e7b648d	gender	user.attribute
dd0ad071-2166-4380-8d92-4d3e8e7b648d	true	id.token.claim
dd0ad071-2166-4380-8d92-4d3e8e7b648d	true	access.token.claim
dd0ad071-2166-4380-8d92-4d3e8e7b648d	gender	claim.name
dd0ad071-2166-4380-8d92-4d3e8e7b648d	String	jsonType.label
fbd992e5-6526-4b9b-a9bb-1dd62c62c4ac	true	introspection.token.claim
fbd992e5-6526-4b9b-a9bb-1dd62c62c4ac	true	userinfo.token.claim
fbd992e5-6526-4b9b-a9bb-1dd62c62c4ac	picture	user.attribute
fbd992e5-6526-4b9b-a9bb-1dd62c62c4ac	true	id.token.claim
fbd992e5-6526-4b9b-a9bb-1dd62c62c4ac	true	access.token.claim
fbd992e5-6526-4b9b-a9bb-1dd62c62c4ac	picture	claim.name
fbd992e5-6526-4b9b-a9bb-1dd62c62c4ac	String	jsonType.label
06901a48-3ef5-437b-988a-9b886b42bd67	true	introspection.token.claim
06901a48-3ef5-437b-988a-9b886b42bd67	true	userinfo.token.claim
06901a48-3ef5-437b-988a-9b886b42bd67	email	user.attribute
06901a48-3ef5-437b-988a-9b886b42bd67	true	id.token.claim
06901a48-3ef5-437b-988a-9b886b42bd67	true	access.token.claim
06901a48-3ef5-437b-988a-9b886b42bd67	email	claim.name
06901a48-3ef5-437b-988a-9b886b42bd67	String	jsonType.label
8a5e8b29-a29e-46b5-90a6-94633cfd464a	true	introspection.token.claim
8a5e8b29-a29e-46b5-90a6-94633cfd464a	true	userinfo.token.claim
8a5e8b29-a29e-46b5-90a6-94633cfd464a	emailVerified	user.attribute
8a5e8b29-a29e-46b5-90a6-94633cfd464a	true	id.token.claim
8a5e8b29-a29e-46b5-90a6-94633cfd464a	true	access.token.claim
8a5e8b29-a29e-46b5-90a6-94633cfd464a	email_verified	claim.name
8a5e8b29-a29e-46b5-90a6-94633cfd464a	boolean	jsonType.label
534f87c2-6bd2-4f64-af4c-79cbe5e3f9b8	formatted	user.attribute.formatted
534f87c2-6bd2-4f64-af4c-79cbe5e3f9b8	country	user.attribute.country
534f87c2-6bd2-4f64-af4c-79cbe5e3f9b8	true	introspection.token.claim
534f87c2-6bd2-4f64-af4c-79cbe5e3f9b8	postal_code	user.attribute.postal_code
534f87c2-6bd2-4f64-af4c-79cbe5e3f9b8	true	userinfo.token.claim
534f87c2-6bd2-4f64-af4c-79cbe5e3f9b8	street	user.attribute.street
534f87c2-6bd2-4f64-af4c-79cbe5e3f9b8	true	id.token.claim
534f87c2-6bd2-4f64-af4c-79cbe5e3f9b8	region	user.attribute.region
534f87c2-6bd2-4f64-af4c-79cbe5e3f9b8	true	access.token.claim
534f87c2-6bd2-4f64-af4c-79cbe5e3f9b8	locality	user.attribute.locality
543b1b4f-b0af-47e0-b775-404f2d7b63aa	true	introspection.token.claim
543b1b4f-b0af-47e0-b775-404f2d7b63aa	true	userinfo.token.claim
543b1b4f-b0af-47e0-b775-404f2d7b63aa	phoneNumber	user.attribute
543b1b4f-b0af-47e0-b775-404f2d7b63aa	true	id.token.claim
543b1b4f-b0af-47e0-b775-404f2d7b63aa	true	access.token.claim
543b1b4f-b0af-47e0-b775-404f2d7b63aa	phone_number	claim.name
543b1b4f-b0af-47e0-b775-404f2d7b63aa	String	jsonType.label
78c4f2da-e724-4835-82e3-174ee16f3767	true	introspection.token.claim
78c4f2da-e724-4835-82e3-174ee16f3767	true	userinfo.token.claim
78c4f2da-e724-4835-82e3-174ee16f3767	phoneNumberVerified	user.attribute
78c4f2da-e724-4835-82e3-174ee16f3767	true	id.token.claim
78c4f2da-e724-4835-82e3-174ee16f3767	true	access.token.claim
78c4f2da-e724-4835-82e3-174ee16f3767	phone_number_verified	claim.name
78c4f2da-e724-4835-82e3-174ee16f3767	boolean	jsonType.label
2ef85bd8-2808-44c4-a804-2561cd6c5d34	true	introspection.token.claim
2ef85bd8-2808-44c4-a804-2561cd6c5d34	true	access.token.claim
5393b5b2-db3d-49db-b24e-83aa5bbcca43	true	introspection.token.claim
5393b5b2-db3d-49db-b24e-83aa5bbcca43	true	multivalued
5393b5b2-db3d-49db-b24e-83aa5bbcca43	foo	user.attribute
5393b5b2-db3d-49db-b24e-83aa5bbcca43	true	access.token.claim
5393b5b2-db3d-49db-b24e-83aa5bbcca43	resource_access.${client_id}.roles	claim.name
5393b5b2-db3d-49db-b24e-83aa5bbcca43	String	jsonType.label
8f61003b-74d9-4cbc-b8bf-ef5b06cae8ef	true	introspection.token.claim
8f61003b-74d9-4cbc-b8bf-ef5b06cae8ef	true	multivalued
8f61003b-74d9-4cbc-b8bf-ef5b06cae8ef	foo	user.attribute
8f61003b-74d9-4cbc-b8bf-ef5b06cae8ef	true	access.token.claim
8f61003b-74d9-4cbc-b8bf-ef5b06cae8ef	realm_access.roles	claim.name
8f61003b-74d9-4cbc-b8bf-ef5b06cae8ef	String	jsonType.label
f64db3b6-a89f-4c9f-87b2-d96405a27181	true	introspection.token.claim
f64db3b6-a89f-4c9f-87b2-d96405a27181	true	access.token.claim
633b8d3b-9208-4052-b1c7-aabe705f9479	true	introspection.token.claim
633b8d3b-9208-4052-b1c7-aabe705f9479	true	userinfo.token.claim
633b8d3b-9208-4052-b1c7-aabe705f9479	username	user.attribute
633b8d3b-9208-4052-b1c7-aabe705f9479	true	id.token.claim
633b8d3b-9208-4052-b1c7-aabe705f9479	true	access.token.claim
633b8d3b-9208-4052-b1c7-aabe705f9479	upn	claim.name
633b8d3b-9208-4052-b1c7-aabe705f9479	String	jsonType.label
ff0206da-8501-4078-9646-ede3f2304f60	true	introspection.token.claim
ff0206da-8501-4078-9646-ede3f2304f60	true	multivalued
ff0206da-8501-4078-9646-ede3f2304f60	foo	user.attribute
ff0206da-8501-4078-9646-ede3f2304f60	true	id.token.claim
ff0206da-8501-4078-9646-ede3f2304f60	true	access.token.claim
ff0206da-8501-4078-9646-ede3f2304f60	groups	claim.name
ff0206da-8501-4078-9646-ede3f2304f60	String	jsonType.label
43c2d1a7-c647-43c3-9457-e3107e634f02	true	introspection.token.claim
43c2d1a7-c647-43c3-9457-e3107e634f02	true	id.token.claim
43c2d1a7-c647-43c3-9457-e3107e634f02	true	access.token.claim
134421d6-1290-496b-ad09-b0a887b806da	AUTH_TIME	user.session.note
134421d6-1290-496b-ad09-b0a887b806da	true	introspection.token.claim
134421d6-1290-496b-ad09-b0a887b806da	true	id.token.claim
134421d6-1290-496b-ad09-b0a887b806da	true	access.token.claim
134421d6-1290-496b-ad09-b0a887b806da	auth_time	claim.name
134421d6-1290-496b-ad09-b0a887b806da	long	jsonType.label
144c8b70-d272-4c4c-bf3f-10ff7edfa8f6	true	introspection.token.claim
144c8b70-d272-4c4c-bf3f-10ff7edfa8f6	true	access.token.claim
1a10382f-1460-4665-858f-4dd49bdc8077	clientHost	user.session.note
1a10382f-1460-4665-858f-4dd49bdc8077	true	introspection.token.claim
1a10382f-1460-4665-858f-4dd49bdc8077	true	id.token.claim
1a10382f-1460-4665-858f-4dd49bdc8077	true	access.token.claim
1a10382f-1460-4665-858f-4dd49bdc8077	clientHost	claim.name
1a10382f-1460-4665-858f-4dd49bdc8077	String	jsonType.label
3d61dff0-76fa-4e86-b922-903fe4d62ff7	clientAddress	user.session.note
3d61dff0-76fa-4e86-b922-903fe4d62ff7	true	introspection.token.claim
3d61dff0-76fa-4e86-b922-903fe4d62ff7	true	id.token.claim
3d61dff0-76fa-4e86-b922-903fe4d62ff7	true	access.token.claim
3d61dff0-76fa-4e86-b922-903fe4d62ff7	clientAddress	claim.name
3d61dff0-76fa-4e86-b922-903fe4d62ff7	String	jsonType.label
6ca113d3-dd50-4749-8604-1291e659025f	client_id	user.session.note
6ca113d3-dd50-4749-8604-1291e659025f	true	introspection.token.claim
6ca113d3-dd50-4749-8604-1291e659025f	true	id.token.claim
6ca113d3-dd50-4749-8604-1291e659025f	true	access.token.claim
6ca113d3-dd50-4749-8604-1291e659025f	client_id	claim.name
6ca113d3-dd50-4749-8604-1291e659025f	String	jsonType.label
7b679bcb-8356-42c8-b058-b778952f5ead	true	introspection.token.claim
7b679bcb-8356-42c8-b058-b778952f5ead	true	multivalued
7b679bcb-8356-42c8-b058-b778952f5ead	true	id.token.claim
7b679bcb-8356-42c8-b058-b778952f5ead	true	access.token.claim
7b679bcb-8356-42c8-b058-b778952f5ead	organization	claim.name
7b679bcb-8356-42c8-b058-b778952f5ead	String	jsonType.label
\.


--
-- Data for Name: realm; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.realm (id, access_code_lifespan, user_action_lifespan, access_token_lifespan, account_theme, admin_theme, email_theme, enabled, events_enabled, events_expiration, login_theme, name, not_before, password_policy, registration_allowed, remember_me, reset_password_allowed, social, ssl_required, sso_idle_timeout, sso_max_lifespan, update_profile_on_soc_login, verify_email, master_admin_client, login_lifespan, internationalization_enabled, default_locale, reg_email_as_username, admin_events_enabled, admin_events_details_enabled, edit_username_allowed, otp_policy_counter, otp_policy_window, otp_policy_period, otp_policy_digits, otp_policy_alg, otp_policy_type, browser_flow, registration_flow, direct_grant_flow, reset_credentials_flow, client_auth_flow, offline_session_idle_timeout, revoke_refresh_token, access_token_life_implicit, login_with_email_allowed, duplicate_emails_allowed, docker_auth_flow, refresh_token_max_reuse, allow_user_managed_access, sso_max_lifespan_remember_me, sso_idle_timeout_remember_me, default_role) FROM stdin;
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	60	300	60	\N	\N	\N	t	f	0	\N	master	0	\N	f	f	f	f	EXTERNAL	1800	36000	f	f	9e64fa07-4221-4426-b056-6d4e83cc7021	1800	f	\N	f	f	f	f	0	1	30	6	HmacSHA1	totp	e8e9d2ff-cf69-44b3-86f4-ea9cce5acea5	c4f5bc43-fefb-4647-a4de-5c6e1a111d0c	abce7f50-bcfc-47fc-a115-a55caa56340c	cb636a91-3068-4711-95c7-981560c9f403	9fb38cd7-2140-4d5f-b1ef-06e83a5d1189	2592000	f	900	t	f	ccf84a7a-d7a7-473b-95cd-ca58ea113743	0	f	0	0	5f661961-94d9-49fb-9627-50430fff0145
\.


--
-- Data for Name: realm_attribute; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.realm_attribute (name, realm_id, value) FROM stdin;
_browser_header.contentSecurityPolicyReportOnly	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	
_browser_header.xContentTypeOptions	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	nosniff
_browser_header.referrerPolicy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	no-referrer
_browser_header.xRobotsTag	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	none
_browser_header.xFrameOptions	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	SAMEORIGIN
_browser_header.contentSecurityPolicy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	frame-src 'self'; frame-ancestors 'self'; object-src 'none';
_browser_header.strictTransportSecurity	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	max-age=31536000; includeSubDomains
bruteForceProtected	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	false
permanentLockout	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	false
maxTemporaryLockouts	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	0
bruteForceStrategy	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	MULTIPLE
maxFailureWaitSeconds	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	900
minimumQuickLoginWaitSeconds	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	60
waitIncrementSeconds	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	60
quickLoginCheckMilliSeconds	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	1000
maxDeltaTimeSeconds	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	43200
failureFactor	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	30
maxSecondaryAuthFailures	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	0
realmReusableOtpCode	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	false
firstBrokerLoginFlowId	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	2264a509-60f6-4146-894d-e4dc1cec61a3
displayName	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	Keycloak
displayNameHtml	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	<div class="kc-logo-text"><span>Keycloak</span></div>
defaultSignatureAlgorithm	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	RS256
offlineSessionMaxLifespanEnabled	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	false
offlineSessionMaxLifespan	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	5184000
\.


--
-- Data for Name: realm_default_groups; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.realm_default_groups (realm_id, group_id) FROM stdin;
\.


--
-- Data for Name: realm_enabled_event_types; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.realm_enabled_event_types (realm_id, value) FROM stdin;
\.


--
-- Data for Name: realm_events_listeners; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.realm_events_listeners (realm_id, value) FROM stdin;
eb86d28d-cdbb-4987-839a-7f8ed73f98ee	jboss-logging
\.


--
-- Data for Name: realm_localizations; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.realm_localizations (realm_id, locale, texts) FROM stdin;
\.


--
-- Data for Name: realm_required_credential; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.realm_required_credential (type, form_label, input, secret, realm_id) FROM stdin;
password	password	t	t	eb86d28d-cdbb-4987-839a-7f8ed73f98ee
\.


--
-- Data for Name: realm_smtp_config; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.realm_smtp_config (realm_id, value, name) FROM stdin;
\.


--
-- Data for Name: realm_supported_locales; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.realm_supported_locales (realm_id, value) FROM stdin;
\.


--
-- Data for Name: redirect_uris; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.redirect_uris (client_id, value) FROM stdin;
c37bd189-0952-45fc-9b3b-c77f8f6e3cb4	/realms/master/account/*
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	/realms/master/account/*
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	/admin/master/console/*
9be1e3f2-527b-4899-b53c-123a908c9d0d	*
\.


--
-- Data for Name: required_action_config; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.required_action_config (required_action_id, value, name) FROM stdin;
\.


--
-- Data for Name: required_action_provider; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.required_action_provider (id, alias, name, realm_id, enabled, default_action, provider_id, priority) FROM stdin;
b0aca6ad-72ea-4fe4-ba70-b81f6a1aee6e	VERIFY_EMAIL	Verify Email	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	t	f	VERIFY_EMAIL	50
4a917f69-b834-448d-ba19-e879a95fc0a3	UPDATE_PROFILE	Update Profile	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	t	f	UPDATE_PROFILE	40
e254322d-a9c7-42c4-9974-6b2c1bff0e00	CONFIGURE_TOTP	Configure OTP	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	t	f	CONFIGURE_TOTP	10
092a518a-712a-4f51-ac9f-84c8babb6da6	UPDATE_PASSWORD	Update Password	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	t	f	UPDATE_PASSWORD	30
8c23350f-df39-45e6-9733-418b3159b264	TERMS_AND_CONDITIONS	Terms and Conditions	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	f	f	TERMS_AND_CONDITIONS	20
2b203afb-1da2-460e-aa6a-8cac9dd9cb62	delete_account	Delete Account	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	f	f	delete_account	60
e0ab377c-d5fe-4525-a531-5c5312646ee8	delete_credential	Delete Credential	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	t	f	delete_credential	110
10a502cc-0470-446e-97f8-e270d88490c6	update_user_locale	Update User Locale	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	t	f	update_user_locale	1000
348c3008-b79c-46b7-b67b-f0f676ac7335	UPDATE_EMAIL	Update Email	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	f	f	UPDATE_EMAIL	70
dccb06d5-3259-410d-ae0a-ffd2257fe9ba	CONFIGURE_RECOVERY_AUTHN_CODES	Recovery Authentication Codes	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	t	f	CONFIGURE_RECOVERY_AUTHN_CODES	130
cac382e8-2105-46de-9e08-ba28426100e0	webauthn-register	Webauthn Register	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	t	f	webauthn-register	80
749ab828-8ff6-4827-998b-bf824b0eb7f9	webauthn-register-passwordless	Webauthn Register Passwordless	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	t	f	webauthn-register-passwordless	90
bf319ff3-edab-4c8e-b505-8030b04efcba	VERIFY_PROFILE	Verify Profile	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	t	f	VERIFY_PROFILE	100
9dab46fd-790a-435a-bb69-4eec4038ed5e	idp_link	Linking Identity Provider	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	t	f	idp_link	120
\.


--
-- Data for Name: resource_attribute; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.resource_attribute (id, name, value, resource_id) FROM stdin;
\.


--
-- Data for Name: resource_policy; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.resource_policy (resource_id, policy_id) FROM stdin;
\.


--
-- Data for Name: resource_scope; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.resource_scope (resource_id, scope_id) FROM stdin;
\.


--
-- Data for Name: resource_server; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.resource_server (id, allow_rs_remote_mgmt, policy_enforce_mode, decision_strategy) FROM stdin;
\.


--
-- Data for Name: resource_server_perm_ticket; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.resource_server_perm_ticket (id, owner, requester, created_timestamp, granted_timestamp, resource_id, scope_id, resource_server_id, policy_id) FROM stdin;
\.


--
-- Data for Name: resource_server_policy; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.resource_server_policy (id, name, description, type, decision_strategy, logic, resource_server_id, owner) FROM stdin;
\.


--
-- Data for Name: resource_server_resource; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.resource_server_resource (id, name, type, icon_uri, owner, resource_server_id, owner_managed_access, display_name) FROM stdin;
\.


--
-- Data for Name: resource_server_scope; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.resource_server_scope (id, name, icon_uri, resource_server_id, display_name) FROM stdin;
\.


--
-- Data for Name: resource_uris; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.resource_uris (resource_id, value) FROM stdin;
\.


--
-- Data for Name: revoked_token; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.revoked_token (id, expire) FROM stdin;
\.


--
-- Data for Name: role_attribute; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.role_attribute (id, role_id, name, value) FROM stdin;
\.


--
-- Data for Name: scope_mapping; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.scope_mapping (client_id, role_id) FROM stdin;
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	681c4058-b23f-4a15-b885-2ac090faa232
e16e1bd2-ccc5-418d-87c3-fc45f25526b7	37b1c2a2-23c5-498c-b596-eb4550ad62fc
\.


--
-- Data for Name: scope_policy; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.scope_policy (scope_id, policy_id) FROM stdin;
\.


--
-- Data for Name: server_config; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.server_config (server_config_key, value, version) FROM stdin;
\.


--
-- Data for Name: user_attribute; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.user_attribute (name, value, user_id, id, long_value_hash, long_value_hash_lower_case, long_value) FROM stdin;
\.


--
-- Data for Name: user_consent; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.user_consent (id, client_id, user_id, created_date, last_updated_date, client_storage_provider, external_client_id) FROM stdin;
\.


--
-- Data for Name: user_consent_client_scope; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.user_consent_client_scope (user_consent_id, scope_id) FROM stdin;
\.


--
-- Data for Name: user_entity; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.user_entity (id, email, email_constraint, email_verified, enabled, federation_link, first_name, last_name, realm_id, username, created_timestamp, service_account_client_link, not_before, last_modified_timestamp) FROM stdin;
f35a80a3-4d9f-4a68-997e-c470f30f2281	admin@admin.com	admin@admin.com	t	t	\N	admin	admin	eb86d28d-cdbb-4987-839a-7f8ed73f98ee	admin	1781917002178	\N	0	1781917002178
\.


--
-- Data for Name: user_federation_config; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.user_federation_config (user_federation_provider_id, value, name) FROM stdin;
\.


--
-- Data for Name: user_federation_mapper; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.user_federation_mapper (id, name, federation_provider_id, federation_mapper_type, realm_id) FROM stdin;
\.


--
-- Data for Name: user_federation_mapper_config; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.user_federation_mapper_config (user_federation_mapper_id, value, name) FROM stdin;
\.


--
-- Data for Name: user_federation_provider; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.user_federation_provider (id, changed_sync_period, display_name, full_sync_period, last_sync, priority, provider_name, realm_id) FROM stdin;
\.


--
-- Data for Name: user_group_membership; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.user_group_membership (group_id, user_id, membership_type) FROM stdin;
\.


--
-- Data for Name: user_required_action; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.user_required_action (user_id, required_action) FROM stdin;
\.


--
-- Data for Name: user_role_mapping; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.user_role_mapping (role_id, user_id) FROM stdin;
5f661961-94d9-49fb-9627-50430fff0145	f35a80a3-4d9f-4a68-997e-c470f30f2281
d74b8256-c52a-45ef-86e8-b8ec3391ee89	f35a80a3-4d9f-4a68-997e-c470f30f2281
5c4f9699-e858-4348-8c6e-7ed861b76b81	f35a80a3-4d9f-4a68-997e-c470f30f2281
10fbb334-c415-430e-9b09-496038b00760	f35a80a3-4d9f-4a68-997e-c470f30f2281
97095622-d040-4d6a-b137-24db9fe89025	f35a80a3-4d9f-4a68-997e-c470f30f2281
\.


--
-- Data for Name: web_origins; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.web_origins (client_id, value) FROM stdin;
89b6f1bc-5ea3-432f-9bd5-9d59c5ebb7f1	+
9be1e3f2-527b-4899-b53c-123a908c9d0d	
\.


--
-- Data for Name: workflow_state; Type: TABLE DATA; Schema: public; Owner: keycloak
--

COPY public.workflow_state (execution_id, resource_id, workflow_id, resource_type, scheduled_step_id, scheduled_step_timestamp) FROM stdin;
\.


--
-- Name: org_domain ORG_DOMAIN_pkey; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.org_domain
    ADD CONSTRAINT "ORG_DOMAIN_pkey" PRIMARY KEY (id, name);


--
-- Name: org ORG_pkey; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.org
    ADD CONSTRAINT "ORG_pkey" PRIMARY KEY (id);


--
-- Name: server_config SERVER_CONFIG_pkey; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.server_config
    ADD CONSTRAINT "SERVER_CONFIG_pkey" PRIMARY KEY (server_config_key);


--
-- Name: keycloak_role UK_J3RWUVD56ONTGSUHOGM184WW2-2; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.keycloak_role
    ADD CONSTRAINT "UK_J3RWUVD56ONTGSUHOGM184WW2-2" UNIQUE (name, client_realm_constraint);


--
-- Name: client_auth_flow_bindings c_cli_flow_bind; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_auth_flow_bindings
    ADD CONSTRAINT c_cli_flow_bind PRIMARY KEY (client_id, binding_name);


--
-- Name: client_scope_client c_cli_scope_bind; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_scope_client
    ADD CONSTRAINT c_cli_scope_bind PRIMARY KEY (client_id, scope_id);


--
-- Name: client_initial_access cnstr_client_init_acc_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_initial_access
    ADD CONSTRAINT cnstr_client_init_acc_pk PRIMARY KEY (id);


--
-- Name: realm_default_groups con_group_id_def_groups; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_default_groups
    ADD CONSTRAINT con_group_id_def_groups UNIQUE (group_id);


--
-- Name: broker_link constr_broker_link_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.broker_link
    ADD CONSTRAINT constr_broker_link_pk PRIMARY KEY (identity_provider, user_id);


--
-- Name: component_config constr_component_config_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.component_config
    ADD CONSTRAINT constr_component_config_pk PRIMARY KEY (id);


--
-- Name: component constr_component_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.component
    ADD CONSTRAINT constr_component_pk PRIMARY KEY (id);


--
-- Name: fed_user_required_action constr_fed_required_action; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.fed_user_required_action
    ADD CONSTRAINT constr_fed_required_action PRIMARY KEY (required_action, user_id);


--
-- Name: fed_user_attribute constr_fed_user_attr_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.fed_user_attribute
    ADD CONSTRAINT constr_fed_user_attr_pk PRIMARY KEY (id);


--
-- Name: fed_user_consent constr_fed_user_consent_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.fed_user_consent
    ADD CONSTRAINT constr_fed_user_consent_pk PRIMARY KEY (id);


--
-- Name: fed_user_credential constr_fed_user_cred_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.fed_user_credential
    ADD CONSTRAINT constr_fed_user_cred_pk PRIMARY KEY (id);


--
-- Name: fed_user_group_membership constr_fed_user_group; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.fed_user_group_membership
    ADD CONSTRAINT constr_fed_user_group PRIMARY KEY (group_id, user_id);


--
-- Name: fed_user_role_mapping constr_fed_user_role; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.fed_user_role_mapping
    ADD CONSTRAINT constr_fed_user_role PRIMARY KEY (role_id, user_id);


--
-- Name: federated_user constr_federated_user; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.federated_user
    ADD CONSTRAINT constr_federated_user PRIMARY KEY (id);


--
-- Name: realm_default_groups constr_realm_default_groups; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_default_groups
    ADD CONSTRAINT constr_realm_default_groups PRIMARY KEY (realm_id, group_id);


--
-- Name: realm_enabled_event_types constr_realm_enabl_event_types; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_enabled_event_types
    ADD CONSTRAINT constr_realm_enabl_event_types PRIMARY KEY (realm_id, value);


--
-- Name: realm_events_listeners constr_realm_events_listeners; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_events_listeners
    ADD CONSTRAINT constr_realm_events_listeners PRIMARY KEY (realm_id, value);


--
-- Name: realm_supported_locales constr_realm_supported_locales; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_supported_locales
    ADD CONSTRAINT constr_realm_supported_locales PRIMARY KEY (realm_id, value);


--
-- Name: identity_provider constraint_2b; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.identity_provider
    ADD CONSTRAINT constraint_2b PRIMARY KEY (internal_id);


--
-- Name: client_attributes constraint_3c; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_attributes
    ADD CONSTRAINT constraint_3c PRIMARY KEY (client_id, name);


--
-- Name: event_entity constraint_4; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.event_entity
    ADD CONSTRAINT constraint_4 PRIMARY KEY (id);


--
-- Name: federated_identity constraint_40; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.federated_identity
    ADD CONSTRAINT constraint_40 PRIMARY KEY (identity_provider, user_id);


--
-- Name: realm constraint_4a; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm
    ADD CONSTRAINT constraint_4a PRIMARY KEY (id);


--
-- Name: user_federation_provider constraint_5c; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_federation_provider
    ADD CONSTRAINT constraint_5c PRIMARY KEY (id);


--
-- Name: client constraint_7; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client
    ADD CONSTRAINT constraint_7 PRIMARY KEY (id);


--
-- Name: scope_mapping constraint_81; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.scope_mapping
    ADD CONSTRAINT constraint_81 PRIMARY KEY (client_id, role_id);


--
-- Name: client_node_registrations constraint_84; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_node_registrations
    ADD CONSTRAINT constraint_84 PRIMARY KEY (client_id, name);


--
-- Name: realm_attribute constraint_9; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_attribute
    ADD CONSTRAINT constraint_9 PRIMARY KEY (name, realm_id);


--
-- Name: realm_required_credential constraint_92; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_required_credential
    ADD CONSTRAINT constraint_92 PRIMARY KEY (realm_id, type);


--
-- Name: keycloak_role constraint_a; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.keycloak_role
    ADD CONSTRAINT constraint_a PRIMARY KEY (id);


--
-- Name: admin_event_entity constraint_admin_event_entity; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.admin_event_entity
    ADD CONSTRAINT constraint_admin_event_entity PRIMARY KEY (id);


--
-- Name: authenticator_config_entry constraint_auth_cfg_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.authenticator_config_entry
    ADD CONSTRAINT constraint_auth_cfg_pk PRIMARY KEY (authenticator_id, name);


--
-- Name: authentication_execution constraint_auth_exec_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.authentication_execution
    ADD CONSTRAINT constraint_auth_exec_pk PRIMARY KEY (id);


--
-- Name: authentication_flow constraint_auth_flow_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.authentication_flow
    ADD CONSTRAINT constraint_auth_flow_pk PRIMARY KEY (id);


--
-- Name: authenticator_config constraint_auth_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.authenticator_config
    ADD CONSTRAINT constraint_auth_pk PRIMARY KEY (id);


--
-- Name: user_role_mapping constraint_c; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_role_mapping
    ADD CONSTRAINT constraint_c PRIMARY KEY (role_id, user_id);


--
-- Name: composite_role constraint_composite_role; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.composite_role
    ADD CONSTRAINT constraint_composite_role PRIMARY KEY (composite, child_role);


--
-- Name: identity_provider_config constraint_d; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.identity_provider_config
    ADD CONSTRAINT constraint_d PRIMARY KEY (identity_provider_id, name);


--
-- Name: policy_config constraint_dpc; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.policy_config
    ADD CONSTRAINT constraint_dpc PRIMARY KEY (policy_id, name);


--
-- Name: realm_smtp_config constraint_e; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_smtp_config
    ADD CONSTRAINT constraint_e PRIMARY KEY (realm_id, name);


--
-- Name: credential constraint_f; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.credential
    ADD CONSTRAINT constraint_f PRIMARY KEY (id);


--
-- Name: user_federation_config constraint_f9; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_federation_config
    ADD CONSTRAINT constraint_f9 PRIMARY KEY (user_federation_provider_id, name);


--
-- Name: resource_server_perm_ticket constraint_fapmt; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_perm_ticket
    ADD CONSTRAINT constraint_fapmt PRIMARY KEY (id);


--
-- Name: resource_server_resource constraint_farsr; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_resource
    ADD CONSTRAINT constraint_farsr PRIMARY KEY (id);


--
-- Name: resource_server_policy constraint_farsrp; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_policy
    ADD CONSTRAINT constraint_farsrp PRIMARY KEY (id);


--
-- Name: associated_policy constraint_farsrpap; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.associated_policy
    ADD CONSTRAINT constraint_farsrpap PRIMARY KEY (policy_id, associated_policy_id);


--
-- Name: resource_policy constraint_farsrpp; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_policy
    ADD CONSTRAINT constraint_farsrpp PRIMARY KEY (resource_id, policy_id);


--
-- Name: resource_server_scope constraint_farsrs; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_scope
    ADD CONSTRAINT constraint_farsrs PRIMARY KEY (id);


--
-- Name: resource_scope constraint_farsrsp; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_scope
    ADD CONSTRAINT constraint_farsrsp PRIMARY KEY (resource_id, scope_id);


--
-- Name: scope_policy constraint_farsrsps; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.scope_policy
    ADD CONSTRAINT constraint_farsrsps PRIMARY KEY (scope_id, policy_id);


--
-- Name: user_entity constraint_fb; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_entity
    ADD CONSTRAINT constraint_fb PRIMARY KEY (id);


--
-- Name: user_federation_mapper_config constraint_fedmapper_cfg_pm; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_federation_mapper_config
    ADD CONSTRAINT constraint_fedmapper_cfg_pm PRIMARY KEY (user_federation_mapper_id, name);


--
-- Name: user_federation_mapper constraint_fedmapperpm; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_federation_mapper
    ADD CONSTRAINT constraint_fedmapperpm PRIMARY KEY (id);


--
-- Name: fed_user_consent_cl_scope constraint_fgrntcsnt_clsc_pm; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.fed_user_consent_cl_scope
    ADD CONSTRAINT constraint_fgrntcsnt_clsc_pm PRIMARY KEY (user_consent_id, scope_id);


--
-- Name: user_consent_client_scope constraint_grntcsnt_clsc_pm; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_consent_client_scope
    ADD CONSTRAINT constraint_grntcsnt_clsc_pm PRIMARY KEY (user_consent_id, scope_id);


--
-- Name: user_consent constraint_grntcsnt_pm; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_consent
    ADD CONSTRAINT constraint_grntcsnt_pm PRIMARY KEY (id);


--
-- Name: keycloak_group constraint_group; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.keycloak_group
    ADD CONSTRAINT constraint_group PRIMARY KEY (id);


--
-- Name: group_attribute constraint_group_attribute_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.group_attribute
    ADD CONSTRAINT constraint_group_attribute_pk PRIMARY KEY (id);


--
-- Name: group_role_mapping constraint_group_role; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.group_role_mapping
    ADD CONSTRAINT constraint_group_role PRIMARY KEY (role_id, group_id);


--
-- Name: identity_provider_mapper constraint_idpm; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.identity_provider_mapper
    ADD CONSTRAINT constraint_idpm PRIMARY KEY (id);


--
-- Name: idp_mapper_config constraint_idpmconfig; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.idp_mapper_config
    ADD CONSTRAINT constraint_idpmconfig PRIMARY KEY (idp_mapper_id, name);


--
-- Name: jgroups_ping constraint_jgroups_ping; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.jgroups_ping
    ADD CONSTRAINT constraint_jgroups_ping PRIMARY KEY (address);


--
-- Name: migration_model constraint_migmod; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.migration_model
    ADD CONSTRAINT constraint_migmod PRIMARY KEY (id);


--
-- Name: offline_client_session constraint_offl_cl_ses_pk3; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.offline_client_session
    ADD CONSTRAINT constraint_offl_cl_ses_pk3 PRIMARY KEY (user_session_id, client_id, client_storage_provider, external_client_id, offline_flag);


--
-- Name: offline_user_session constraint_offl_us_ses_pk2; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.offline_user_session
    ADD CONSTRAINT constraint_offl_us_ses_pk2 PRIMARY KEY (user_session_id, offline_flag);


--
-- Name: org_invitation constraint_org_invitation; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.org_invitation
    ADD CONSTRAINT constraint_org_invitation PRIMARY KEY (id);


--
-- Name: protocol_mapper constraint_pcm; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.protocol_mapper
    ADD CONSTRAINT constraint_pcm PRIMARY KEY (id);


--
-- Name: protocol_mapper_config constraint_pmconfig; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.protocol_mapper_config
    ADD CONSTRAINT constraint_pmconfig PRIMARY KEY (protocol_mapper_id, name);


--
-- Name: redirect_uris constraint_redirect_uris; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.redirect_uris
    ADD CONSTRAINT constraint_redirect_uris PRIMARY KEY (client_id, value);


--
-- Name: required_action_config constraint_req_act_cfg_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.required_action_config
    ADD CONSTRAINT constraint_req_act_cfg_pk PRIMARY KEY (required_action_id, name);


--
-- Name: required_action_provider constraint_req_act_prv_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.required_action_provider
    ADD CONSTRAINT constraint_req_act_prv_pk PRIMARY KEY (id);


--
-- Name: user_required_action constraint_required_action; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_required_action
    ADD CONSTRAINT constraint_required_action PRIMARY KEY (required_action, user_id);


--
-- Name: resource_uris constraint_resour_uris_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_uris
    ADD CONSTRAINT constraint_resour_uris_pk PRIMARY KEY (resource_id, value);


--
-- Name: role_attribute constraint_role_attribute_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.role_attribute
    ADD CONSTRAINT constraint_role_attribute_pk PRIMARY KEY (id);


--
-- Name: revoked_token constraint_rt; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.revoked_token
    ADD CONSTRAINT constraint_rt PRIMARY KEY (id);


--
-- Name: user_attribute constraint_user_attribute_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_attribute
    ADD CONSTRAINT constraint_user_attribute_pk PRIMARY KEY (id);


--
-- Name: user_group_membership constraint_user_group; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_group_membership
    ADD CONSTRAINT constraint_user_group PRIMARY KEY (group_id, user_id);


--
-- Name: web_origins constraint_web_origins; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.web_origins
    ADD CONSTRAINT constraint_web_origins PRIMARY KEY (client_id, value);


--
-- Name: databasechangeloglock databasechangeloglock_pkey; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.databasechangeloglock
    ADD CONSTRAINT databasechangeloglock_pkey PRIMARY KEY (id);


--
-- Name: client_scope_attributes pk_cl_tmpl_attr; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_scope_attributes
    ADD CONSTRAINT pk_cl_tmpl_attr PRIMARY KEY (scope_id, name);


--
-- Name: client_scope pk_cli_template; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_scope
    ADD CONSTRAINT pk_cli_template PRIMARY KEY (id);


--
-- Name: resource_server pk_resource_server; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server
    ADD CONSTRAINT pk_resource_server PRIMARY KEY (id);


--
-- Name: client_scope_role_mapping pk_template_scope; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_scope_role_mapping
    ADD CONSTRAINT pk_template_scope PRIMARY KEY (scope_id, role_id);


--
-- Name: workflow_state pk_workflow_state; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.workflow_state
    ADD CONSTRAINT pk_workflow_state PRIMARY KEY (execution_id);


--
-- Name: default_client_scope r_def_cli_scope_bind; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.default_client_scope
    ADD CONSTRAINT r_def_cli_scope_bind PRIMARY KEY (realm_id, scope_id);


--
-- Name: realm_localizations realm_localizations_pkey; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_localizations
    ADD CONSTRAINT realm_localizations_pkey PRIMARY KEY (realm_id, locale);


--
-- Name: resource_attribute res_attr_pk; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_attribute
    ADD CONSTRAINT res_attr_pk PRIMARY KEY (id);


--
-- Name: keycloak_group sibling_names; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.keycloak_group
    ADD CONSTRAINT sibling_names UNIQUE (realm_id, parent_group, name);


--
-- Name: identity_provider uk_2daelwnibji49avxsrtuf6xj33; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.identity_provider
    ADD CONSTRAINT uk_2daelwnibji49avxsrtuf6xj33 UNIQUE (provider_alias, realm_id);


--
-- Name: client uk_b71cjlbenv945rb6gcon438at; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client
    ADD CONSTRAINT uk_b71cjlbenv945rb6gcon438at UNIQUE (realm_id, client_id);


--
-- Name: client_scope uk_cli_scope; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_scope
    ADD CONSTRAINT uk_cli_scope UNIQUE (realm_id, name);


--
-- Name: user_entity uk_dykn684sl8up1crfei6eckhd7; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_entity
    ADD CONSTRAINT uk_dykn684sl8up1crfei6eckhd7 UNIQUE (realm_id, email_constraint);


--
-- Name: user_consent uk_external_consent; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_consent
    ADD CONSTRAINT uk_external_consent UNIQUE (client_storage_provider, external_client_id, user_id);


--
-- Name: resource_server_resource uk_frsr6t700s9v50bu18ws5ha6; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_resource
    ADD CONSTRAINT uk_frsr6t700s9v50bu18ws5ha6 UNIQUE (name, owner, resource_server_id);


--
-- Name: resource_server_perm_ticket uk_frsr6t700s9v50bu18ws5pmt; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_perm_ticket
    ADD CONSTRAINT uk_frsr6t700s9v50bu18ws5pmt UNIQUE (owner, requester, resource_server_id, resource_id, scope_id);


--
-- Name: resource_server_policy uk_frsrpt700s9v50bu18ws5ha6; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_policy
    ADD CONSTRAINT uk_frsrpt700s9v50bu18ws5ha6 UNIQUE (name, resource_server_id);


--
-- Name: resource_server_scope uk_frsrst700s9v50bu18ws5ha6; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_scope
    ADD CONSTRAINT uk_frsrst700s9v50bu18ws5ha6 UNIQUE (name, resource_server_id);


--
-- Name: user_consent uk_local_consent; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_consent
    ADD CONSTRAINT uk_local_consent UNIQUE (client_id, user_id);


--
-- Name: migration_model uk_migration_update_time; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.migration_model
    ADD CONSTRAINT uk_migration_update_time UNIQUE (update_time);


--
-- Name: migration_model uk_migration_version; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.migration_model
    ADD CONSTRAINT uk_migration_version UNIQUE (version);


--
-- Name: org uk_org_alias; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.org
    ADD CONSTRAINT uk_org_alias UNIQUE (realm_id, alias);


--
-- Name: org uk_org_group; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.org
    ADD CONSTRAINT uk_org_group UNIQUE (group_id);


--
-- Name: org_invitation uk_org_invitation_email; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.org_invitation
    ADD CONSTRAINT uk_org_invitation_email UNIQUE (organization_id, email);


--
-- Name: org uk_org_name; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.org
    ADD CONSTRAINT uk_org_name UNIQUE (realm_id, name);


--
-- Name: realm uk_orvsdmla56612eaefiq6wl5oi; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm
    ADD CONSTRAINT uk_orvsdmla56612eaefiq6wl5oi UNIQUE (name);


--
-- Name: user_entity uk_ru8tt6t700s9v50bu18ws5ha6; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_entity
    ADD CONSTRAINT uk_ru8tt6t700s9v50bu18ws5ha6 UNIQUE (realm_id, username);


--
-- Name: workflow_state uq_workflow_resource; Type: CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.workflow_state
    ADD CONSTRAINT uq_workflow_resource UNIQUE (workflow_id, resource_id);


--
-- Name: fed_user_attr_long_values; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX fed_user_attr_long_values ON public.fed_user_attribute USING btree (long_value_hash, name);


--
-- Name: fed_user_attr_long_values_lower_case; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX fed_user_attr_long_values_lower_case ON public.fed_user_attribute USING btree (long_value_hash_lower_case, name);


--
-- Name: idx_admin_event_time; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_admin_event_time ON public.admin_event_entity USING btree (realm_id, admin_event_time);


--
-- Name: idx_assoc_pol_assoc_pol_id; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_assoc_pol_assoc_pol_id ON public.associated_policy USING btree (associated_policy_id);


--
-- Name: idx_auth_config_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_auth_config_realm ON public.authenticator_config USING btree (realm_id);


--
-- Name: idx_auth_exec_flow; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_auth_exec_flow ON public.authentication_execution USING btree (flow_id);


--
-- Name: idx_auth_exec_realm_flow; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_auth_exec_realm_flow ON public.authentication_execution USING btree (realm_id, flow_id);


--
-- Name: idx_auth_flow_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_auth_flow_realm ON public.authentication_flow USING btree (realm_id);


--
-- Name: idx_broker_link_identity_provider; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_broker_link_identity_provider ON public.broker_link USING btree (realm_id, identity_provider, broker_user_id);


--
-- Name: idx_broker_link_user_id; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_broker_link_user_id ON public.broker_link USING btree (user_id);


--
-- Name: idx_cl_clscope; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_cl_clscope ON public.client_scope_client USING btree (scope_id);


--
-- Name: idx_client_att_by_name_value; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_client_att_by_name_value ON public.client_attributes USING btree (name, substr(value, 1, 255));


--
-- Name: idx_client_id; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_client_id ON public.client USING btree (client_id);


--
-- Name: idx_client_init_acc_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_client_init_acc_realm ON public.client_initial_access USING btree (realm_id);


--
-- Name: idx_clscope_attrs; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_clscope_attrs ON public.client_scope_attributes USING btree (scope_id);


--
-- Name: idx_clscope_cl; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_clscope_cl ON public.client_scope_client USING btree (client_id);


--
-- Name: idx_clscope_protmap; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_clscope_protmap ON public.protocol_mapper USING btree (client_scope_id);


--
-- Name: idx_clscope_role; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_clscope_role ON public.client_scope_role_mapping USING btree (scope_id);


--
-- Name: idx_compo_config_compo; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_compo_config_compo ON public.component_config USING btree (component_id);


--
-- Name: idx_component_provider_type; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_component_provider_type ON public.component USING btree (provider_type);


--
-- Name: idx_component_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_component_realm ON public.component USING btree (realm_id);


--
-- Name: idx_composite; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_composite ON public.composite_role USING btree (composite);


--
-- Name: idx_composite_child; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_composite_child ON public.composite_role USING btree (child_role);


--
-- Name: idx_defcls_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_defcls_realm ON public.default_client_scope USING btree (realm_id);


--
-- Name: idx_defcls_scope; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_defcls_scope ON public.default_client_scope USING btree (scope_id);


--
-- Name: idx_event_entity_user_id_type; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_event_entity_user_id_type ON public.event_entity USING btree (user_id, type, event_time);


--
-- Name: idx_event_time; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_event_time ON public.event_entity USING btree (realm_id, event_time);


--
-- Name: idx_fedidentity_feduser; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fedidentity_feduser ON public.federated_identity USING btree (federated_user_id);


--
-- Name: idx_fedidentity_user; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fedidentity_user ON public.federated_identity USING btree (user_id);


--
-- Name: idx_fu_attribute; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fu_attribute ON public.fed_user_attribute USING btree (user_id, realm_id, name);


--
-- Name: idx_fu_cnsnt_ext; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fu_cnsnt_ext ON public.fed_user_consent USING btree (user_id, client_storage_provider, external_client_id);


--
-- Name: idx_fu_consent; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fu_consent ON public.fed_user_consent USING btree (user_id, client_id);


--
-- Name: idx_fu_consent_ru; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fu_consent_ru ON public.fed_user_consent USING btree (realm_id, user_id);


--
-- Name: idx_fu_credential; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fu_credential ON public.fed_user_credential USING btree (user_id, type);


--
-- Name: idx_fu_credential_ru; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fu_credential_ru ON public.fed_user_credential USING btree (realm_id, user_id);


--
-- Name: idx_fu_group_membership; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fu_group_membership ON public.fed_user_group_membership USING btree (user_id, group_id);


--
-- Name: idx_fu_group_membership_ru; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fu_group_membership_ru ON public.fed_user_group_membership USING btree (realm_id, user_id);


--
-- Name: idx_fu_required_action; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fu_required_action ON public.fed_user_required_action USING btree (user_id, required_action);


--
-- Name: idx_fu_required_action_ru; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fu_required_action_ru ON public.fed_user_required_action USING btree (realm_id, user_id);


--
-- Name: idx_fu_role_mapping; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fu_role_mapping ON public.fed_user_role_mapping USING btree (user_id, role_id);


--
-- Name: idx_fu_role_mapping_ru; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_fu_role_mapping_ru ON public.fed_user_role_mapping USING btree (realm_id, user_id);


--
-- Name: idx_group_att_by_name_value; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_group_att_by_name_value ON public.group_attribute USING btree (name, ((value)::character varying(250)));


--
-- Name: idx_group_attr_group; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_group_attr_group ON public.group_attribute USING btree (group_id);


--
-- Name: idx_group_org_id; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_group_org_id ON public.keycloak_group USING btree (org_id);


--
-- Name: idx_group_role_mapp_group; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_group_role_mapp_group ON public.group_role_mapping USING btree (group_id);


--
-- Name: idx_id_prov_mapp_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_id_prov_mapp_realm ON public.identity_provider_mapper USING btree (realm_id);


--
-- Name: idx_ident_prov_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_ident_prov_realm ON public.identity_provider USING btree (realm_id);


--
-- Name: idx_idp_for_login; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_idp_for_login ON public.identity_provider USING btree (realm_id, enabled, link_only, hide_on_login, organization_id);


--
-- Name: idx_idp_realm_org; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_idp_realm_org ON public.identity_provider USING btree (realm_id, organization_id);


--
-- Name: idx_keycloak_role_client; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_keycloak_role_client ON public.keycloak_role USING btree (client);


--
-- Name: idx_keycloak_role_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_keycloak_role_realm ON public.keycloak_role USING btree (realm);


--
-- Name: idx_offline_css_by_client; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_offline_css_by_client ON public.offline_client_session USING btree (client_id, offline_flag) WHERE ((client_id)::text <> 'external'::text);


--
-- Name: idx_offline_css_by_client_and_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_offline_css_by_client_and_realm ON public.offline_client_session USING btree (realm_id, offline_flag, client_id, client_storage_provider, external_client_id);


--
-- Name: idx_offline_css_by_client_storage_provider; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_offline_css_by_client_storage_provider ON public.offline_client_session USING btree (client_storage_provider, external_client_id, offline_flag) WHERE ((client_storage_provider)::text <> 'internal'::text);


--
-- Name: idx_offline_css_by_user_session_and_offline; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_offline_css_by_user_session_and_offline ON public.offline_client_session USING btree (offline_flag, user_session_id);


--
-- Name: idx_offline_uss_by_broker_session_id; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_offline_uss_by_broker_session_id ON public.offline_user_session USING btree (broker_session_id, realm_id);


--
-- Name: idx_offline_uss_by_user; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_offline_uss_by_user ON public.offline_user_session USING btree (user_id, realm_id, offline_flag);


--
-- Name: idx_org_domain_org_id; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_org_domain_org_id ON public.org_domain USING btree (org_id);


--
-- Name: idx_org_invitation_email; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_org_invitation_email ON public.org_invitation USING btree (email);


--
-- Name: idx_org_invitation_expires; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_org_invitation_expires ON public.org_invitation USING btree (expires_at);


--
-- Name: idx_org_invitation_org_id; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_org_invitation_org_id ON public.org_invitation USING btree (organization_id);


--
-- Name: idx_perm_ticket_owner; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_perm_ticket_owner ON public.resource_server_perm_ticket USING btree (owner);


--
-- Name: idx_perm_ticket_requester; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_perm_ticket_requester ON public.resource_server_perm_ticket USING btree (requester);


--
-- Name: idx_protocol_mapper_client; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_protocol_mapper_client ON public.protocol_mapper USING btree (client_id);


--
-- Name: idx_realm_attr_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_realm_attr_realm ON public.realm_attribute USING btree (realm_id);


--
-- Name: idx_realm_clscope; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_realm_clscope ON public.client_scope USING btree (realm_id);


--
-- Name: idx_realm_def_grp_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_realm_def_grp_realm ON public.realm_default_groups USING btree (realm_id);


--
-- Name: idx_realm_evt_list_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_realm_evt_list_realm ON public.realm_events_listeners USING btree (realm_id);


--
-- Name: idx_realm_evt_types_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_realm_evt_types_realm ON public.realm_enabled_event_types USING btree (realm_id);


--
-- Name: idx_realm_master_adm_cli; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_realm_master_adm_cli ON public.realm USING btree (master_admin_client);


--
-- Name: idx_realm_supp_local_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_realm_supp_local_realm ON public.realm_supported_locales USING btree (realm_id);


--
-- Name: idx_redir_uri_client; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_redir_uri_client ON public.redirect_uris USING btree (client_id);


--
-- Name: idx_req_act_prov_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_req_act_prov_realm ON public.required_action_provider USING btree (realm_id);


--
-- Name: idx_res_policy_policy; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_res_policy_policy ON public.resource_policy USING btree (policy_id);


--
-- Name: idx_res_scope_scope; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_res_scope_scope ON public.resource_scope USING btree (scope_id);


--
-- Name: idx_res_serv_pol_res_serv; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_res_serv_pol_res_serv ON public.resource_server_policy USING btree (resource_server_id);


--
-- Name: idx_res_srv_res_res_srv; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_res_srv_res_res_srv ON public.resource_server_resource USING btree (resource_server_id);


--
-- Name: idx_res_srv_scope_res_srv; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_res_srv_scope_res_srv ON public.resource_server_scope USING btree (resource_server_id);


--
-- Name: idx_rev_token_on_expire; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_rev_token_on_expire ON public.revoked_token USING btree (expire);


--
-- Name: idx_role_attribute; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_role_attribute ON public.role_attribute USING btree (role_id);


--
-- Name: idx_role_clscope; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_role_clscope ON public.client_scope_role_mapping USING btree (role_id);


--
-- Name: idx_scope_mapping_role; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_scope_mapping_role ON public.scope_mapping USING btree (role_id);


--
-- Name: idx_scope_policy_policy; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_scope_policy_policy ON public.scope_policy USING btree (policy_id);


--
-- Name: idx_update_time; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_update_time ON public.migration_model USING btree (update_time);


--
-- Name: idx_usconsent_clscope; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_usconsent_clscope ON public.user_consent_client_scope USING btree (user_consent_id);


--
-- Name: idx_usconsent_scope_id; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_usconsent_scope_id ON public.user_consent_client_scope USING btree (scope_id);


--
-- Name: idx_user_attribute; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_user_attribute ON public.user_attribute USING btree (user_id);


--
-- Name: idx_user_attribute_name; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_user_attribute_name ON public.user_attribute USING btree (name, value);


--
-- Name: idx_user_consent; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_user_consent ON public.user_consent USING btree (user_id);


--
-- Name: idx_user_created_timestamp; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_user_created_timestamp ON public.user_entity USING btree (realm_id, created_timestamp);


--
-- Name: idx_user_credential; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_user_credential ON public.credential USING btree (user_id);


--
-- Name: idx_user_email; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_user_email ON public.user_entity USING btree (email);


--
-- Name: idx_user_group_mapping; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_user_group_mapping ON public.user_group_membership USING btree (user_id);


--
-- Name: idx_user_reqactions; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_user_reqactions ON public.user_required_action USING btree (user_id);


--
-- Name: idx_user_role_mapping; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_user_role_mapping ON public.user_role_mapping USING btree (user_id);


--
-- Name: idx_user_service_account; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_user_service_account ON public.user_entity USING btree (realm_id, service_account_client_link);


--
-- Name: idx_user_session_expiration_created; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_user_session_expiration_created ON public.offline_user_session USING btree (realm_id, offline_flag, remember_me, created_on, user_session_id, user_id);


--
-- Name: idx_user_session_expiration_last_refresh; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_user_session_expiration_last_refresh ON public.offline_user_session USING btree (realm_id, offline_flag, remember_me, last_session_refresh, user_session_id, user_id);


--
-- Name: idx_usr_fed_map_fed_prv; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_usr_fed_map_fed_prv ON public.user_federation_mapper USING btree (federation_provider_id);


--
-- Name: idx_usr_fed_map_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_usr_fed_map_realm ON public.user_federation_mapper USING btree (realm_id);


--
-- Name: idx_usr_fed_prv_realm; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_usr_fed_prv_realm ON public.user_federation_provider USING btree (realm_id);


--
-- Name: idx_web_orig_client; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_web_orig_client ON public.web_origins USING btree (client_id);


--
-- Name: idx_workflow_state_provider; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_workflow_state_provider ON public.workflow_state USING btree (resource_id);


--
-- Name: idx_workflow_state_step; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX idx_workflow_state_step ON public.workflow_state USING btree (workflow_id, scheduled_step_id);


--
-- Name: user_attr_long_values; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX user_attr_long_values ON public.user_attribute USING btree (long_value_hash, name);


--
-- Name: user_attr_long_values_lower_case; Type: INDEX; Schema: public; Owner: keycloak
--

CREATE INDEX user_attr_long_values_lower_case ON public.user_attribute USING btree (long_value_hash_lower_case, name);


--
-- Name: identity_provider fk2b4ebc52ae5c3b34; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.identity_provider
    ADD CONSTRAINT fk2b4ebc52ae5c3b34 FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: client_attributes fk3c47c64beacca966; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_attributes
    ADD CONSTRAINT fk3c47c64beacca966 FOREIGN KEY (client_id) REFERENCES public.client(id);


--
-- Name: federated_identity fk404288b92ef007a6; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.federated_identity
    ADD CONSTRAINT fk404288b92ef007a6 FOREIGN KEY (user_id) REFERENCES public.user_entity(id);


--
-- Name: client_node_registrations fk4129723ba992f594; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_node_registrations
    ADD CONSTRAINT fk4129723ba992f594 FOREIGN KEY (client_id) REFERENCES public.client(id);


--
-- Name: redirect_uris fk_1burs8pb4ouj97h5wuppahv9f; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.redirect_uris
    ADD CONSTRAINT fk_1burs8pb4ouj97h5wuppahv9f FOREIGN KEY (client_id) REFERENCES public.client(id);


--
-- Name: user_federation_provider fk_1fj32f6ptolw2qy60cd8n01e8; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_federation_provider
    ADD CONSTRAINT fk_1fj32f6ptolw2qy60cd8n01e8 FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: realm_required_credential fk_5hg65lybevavkqfki3kponh9v; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_required_credential
    ADD CONSTRAINT fk_5hg65lybevavkqfki3kponh9v FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: resource_attribute fk_5hrm2vlf9ql5fu022kqepovbr; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_attribute
    ADD CONSTRAINT fk_5hrm2vlf9ql5fu022kqepovbr FOREIGN KEY (resource_id) REFERENCES public.resource_server_resource(id);


--
-- Name: user_attribute fk_5hrm2vlf9ql5fu043kqepovbr; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_attribute
    ADD CONSTRAINT fk_5hrm2vlf9ql5fu043kqepovbr FOREIGN KEY (user_id) REFERENCES public.user_entity(id);


--
-- Name: user_required_action fk_6qj3w1jw9cvafhe19bwsiuvmd; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_required_action
    ADD CONSTRAINT fk_6qj3w1jw9cvafhe19bwsiuvmd FOREIGN KEY (user_id) REFERENCES public.user_entity(id);


--
-- Name: keycloak_role fk_6vyqfe4cn4wlq8r6kt5vdsj5c; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.keycloak_role
    ADD CONSTRAINT fk_6vyqfe4cn4wlq8r6kt5vdsj5c FOREIGN KEY (realm) REFERENCES public.realm(id);


--
-- Name: realm_smtp_config fk_70ej8xdxgxd0b9hh6180irr0o; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_smtp_config
    ADD CONSTRAINT fk_70ej8xdxgxd0b9hh6180irr0o FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: realm_attribute fk_8shxd6l3e9atqukacxgpffptw; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_attribute
    ADD CONSTRAINT fk_8shxd6l3e9atqukacxgpffptw FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: composite_role fk_a63wvekftu8jo1pnj81e7mce2; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.composite_role
    ADD CONSTRAINT fk_a63wvekftu8jo1pnj81e7mce2 FOREIGN KEY (composite) REFERENCES public.keycloak_role(id);


--
-- Name: authentication_execution fk_auth_exec_flow; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.authentication_execution
    ADD CONSTRAINT fk_auth_exec_flow FOREIGN KEY (flow_id) REFERENCES public.authentication_flow(id);


--
-- Name: authentication_execution fk_auth_exec_realm; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.authentication_execution
    ADD CONSTRAINT fk_auth_exec_realm FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: authentication_flow fk_auth_flow_realm; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.authentication_flow
    ADD CONSTRAINT fk_auth_flow_realm FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: authenticator_config fk_auth_realm; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.authenticator_config
    ADD CONSTRAINT fk_auth_realm FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: user_role_mapping fk_c4fqv34p1mbylloxang7b1q3l; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_role_mapping
    ADD CONSTRAINT fk_c4fqv34p1mbylloxang7b1q3l FOREIGN KEY (user_id) REFERENCES public.user_entity(id);


--
-- Name: client_scope_attributes fk_cl_scope_attr_scope; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_scope_attributes
    ADD CONSTRAINT fk_cl_scope_attr_scope FOREIGN KEY (scope_id) REFERENCES public.client_scope(id);


--
-- Name: client_scope_role_mapping fk_cl_scope_rm_scope; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_scope_role_mapping
    ADD CONSTRAINT fk_cl_scope_rm_scope FOREIGN KEY (scope_id) REFERENCES public.client_scope(id);


--
-- Name: protocol_mapper fk_cli_scope_mapper; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.protocol_mapper
    ADD CONSTRAINT fk_cli_scope_mapper FOREIGN KEY (client_scope_id) REFERENCES public.client_scope(id);


--
-- Name: client_initial_access fk_client_init_acc_realm; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.client_initial_access
    ADD CONSTRAINT fk_client_init_acc_realm FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: component_config fk_component_config; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.component_config
    ADD CONSTRAINT fk_component_config FOREIGN KEY (component_id) REFERENCES public.component(id);


--
-- Name: component fk_component_realm; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.component
    ADD CONSTRAINT fk_component_realm FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: realm_default_groups fk_def_groups_realm; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_default_groups
    ADD CONSTRAINT fk_def_groups_realm FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: user_federation_mapper_config fk_fedmapper_cfg; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_federation_mapper_config
    ADD CONSTRAINT fk_fedmapper_cfg FOREIGN KEY (user_federation_mapper_id) REFERENCES public.user_federation_mapper(id);


--
-- Name: user_federation_mapper fk_fedmapperpm_fedprv; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_federation_mapper
    ADD CONSTRAINT fk_fedmapperpm_fedprv FOREIGN KEY (federation_provider_id) REFERENCES public.user_federation_provider(id);


--
-- Name: user_federation_mapper fk_fedmapperpm_realm; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_federation_mapper
    ADD CONSTRAINT fk_fedmapperpm_realm FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: associated_policy fk_frsr5s213xcx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.associated_policy
    ADD CONSTRAINT fk_frsr5s213xcx4wnkog82ssrfy FOREIGN KEY (associated_policy_id) REFERENCES public.resource_server_policy(id);


--
-- Name: scope_policy fk_frsrasp13xcx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.scope_policy
    ADD CONSTRAINT fk_frsrasp13xcx4wnkog82ssrfy FOREIGN KEY (policy_id) REFERENCES public.resource_server_policy(id);


--
-- Name: resource_server_perm_ticket fk_frsrho213xcx4wnkog82sspmt; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_perm_ticket
    ADD CONSTRAINT fk_frsrho213xcx4wnkog82sspmt FOREIGN KEY (resource_server_id) REFERENCES public.resource_server(id);


--
-- Name: resource_server_resource fk_frsrho213xcx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_resource
    ADD CONSTRAINT fk_frsrho213xcx4wnkog82ssrfy FOREIGN KEY (resource_server_id) REFERENCES public.resource_server(id);


--
-- Name: resource_server_perm_ticket fk_frsrho213xcx4wnkog83sspmt; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_perm_ticket
    ADD CONSTRAINT fk_frsrho213xcx4wnkog83sspmt FOREIGN KEY (resource_id) REFERENCES public.resource_server_resource(id);


--
-- Name: resource_server_perm_ticket fk_frsrho213xcx4wnkog84sspmt; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_perm_ticket
    ADD CONSTRAINT fk_frsrho213xcx4wnkog84sspmt FOREIGN KEY (scope_id) REFERENCES public.resource_server_scope(id);


--
-- Name: associated_policy fk_frsrpas14xcx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.associated_policy
    ADD CONSTRAINT fk_frsrpas14xcx4wnkog82ssrfy FOREIGN KEY (policy_id) REFERENCES public.resource_server_policy(id);


--
-- Name: scope_policy fk_frsrpass3xcx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.scope_policy
    ADD CONSTRAINT fk_frsrpass3xcx4wnkog82ssrfy FOREIGN KEY (scope_id) REFERENCES public.resource_server_scope(id);


--
-- Name: resource_server_perm_ticket fk_frsrpo2128cx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_perm_ticket
    ADD CONSTRAINT fk_frsrpo2128cx4wnkog82ssrfy FOREIGN KEY (policy_id) REFERENCES public.resource_server_policy(id);


--
-- Name: resource_server_policy fk_frsrpo213xcx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_policy
    ADD CONSTRAINT fk_frsrpo213xcx4wnkog82ssrfy FOREIGN KEY (resource_server_id) REFERENCES public.resource_server(id);


--
-- Name: resource_scope fk_frsrpos13xcx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_scope
    ADD CONSTRAINT fk_frsrpos13xcx4wnkog82ssrfy FOREIGN KEY (resource_id) REFERENCES public.resource_server_resource(id);


--
-- Name: resource_policy fk_frsrpos53xcx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_policy
    ADD CONSTRAINT fk_frsrpos53xcx4wnkog82ssrfy FOREIGN KEY (resource_id) REFERENCES public.resource_server_resource(id);


--
-- Name: resource_policy fk_frsrpp213xcx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_policy
    ADD CONSTRAINT fk_frsrpp213xcx4wnkog82ssrfy FOREIGN KEY (policy_id) REFERENCES public.resource_server_policy(id);


--
-- Name: resource_scope fk_frsrps213xcx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_scope
    ADD CONSTRAINT fk_frsrps213xcx4wnkog82ssrfy FOREIGN KEY (scope_id) REFERENCES public.resource_server_scope(id);


--
-- Name: resource_server_scope fk_frsrso213xcx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_server_scope
    ADD CONSTRAINT fk_frsrso213xcx4wnkog82ssrfy FOREIGN KEY (resource_server_id) REFERENCES public.resource_server(id);


--
-- Name: composite_role fk_gr7thllb9lu8q4vqa4524jjy8; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.composite_role
    ADD CONSTRAINT fk_gr7thllb9lu8q4vqa4524jjy8 FOREIGN KEY (child_role) REFERENCES public.keycloak_role(id);


--
-- Name: user_consent_client_scope fk_grntcsnt_clsc_usc; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_consent_client_scope
    ADD CONSTRAINT fk_grntcsnt_clsc_usc FOREIGN KEY (user_consent_id) REFERENCES public.user_consent(id);


--
-- Name: user_consent fk_grntcsnt_user; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_consent
    ADD CONSTRAINT fk_grntcsnt_user FOREIGN KEY (user_id) REFERENCES public.user_entity(id);


--
-- Name: group_attribute fk_group_attribute_group; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.group_attribute
    ADD CONSTRAINT fk_group_attribute_group FOREIGN KEY (group_id) REFERENCES public.keycloak_group(id);


--
-- Name: keycloak_group fk_group_organization; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.keycloak_group
    ADD CONSTRAINT fk_group_organization FOREIGN KEY (org_id) REFERENCES public.org(id);


--
-- Name: group_role_mapping fk_group_role_group; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.group_role_mapping
    ADD CONSTRAINT fk_group_role_group FOREIGN KEY (group_id) REFERENCES public.keycloak_group(id);


--
-- Name: realm_enabled_event_types fk_h846o4h0w8epx5nwedrf5y69j; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_enabled_event_types
    ADD CONSTRAINT fk_h846o4h0w8epx5nwedrf5y69j FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: realm_events_listeners fk_h846o4h0w8epx5nxev9f5y69j; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_events_listeners
    ADD CONSTRAINT fk_h846o4h0w8epx5nxev9f5y69j FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: identity_provider_mapper fk_idpm_realm; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.identity_provider_mapper
    ADD CONSTRAINT fk_idpm_realm FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: idp_mapper_config fk_idpmconfig; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.idp_mapper_config
    ADD CONSTRAINT fk_idpmconfig FOREIGN KEY (idp_mapper_id) REFERENCES public.identity_provider_mapper(id);


--
-- Name: web_origins fk_lojpho213xcx4wnkog82ssrfy; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.web_origins
    ADD CONSTRAINT fk_lojpho213xcx4wnkog82ssrfy FOREIGN KEY (client_id) REFERENCES public.client(id);


--
-- Name: org_invitation fk_org_invitation_org; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.org_invitation
    ADD CONSTRAINT fk_org_invitation_org FOREIGN KEY (organization_id) REFERENCES public.org(id) ON DELETE CASCADE;


--
-- Name: scope_mapping fk_ouse064plmlr732lxjcn1q5f1; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.scope_mapping
    ADD CONSTRAINT fk_ouse064plmlr732lxjcn1q5f1 FOREIGN KEY (client_id) REFERENCES public.client(id);


--
-- Name: protocol_mapper fk_pcm_realm; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.protocol_mapper
    ADD CONSTRAINT fk_pcm_realm FOREIGN KEY (client_id) REFERENCES public.client(id);


--
-- Name: credential fk_pfyr0glasqyl0dei3kl69r6v0; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.credential
    ADD CONSTRAINT fk_pfyr0glasqyl0dei3kl69r6v0 FOREIGN KEY (user_id) REFERENCES public.user_entity(id);


--
-- Name: protocol_mapper_config fk_pmconfig; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.protocol_mapper_config
    ADD CONSTRAINT fk_pmconfig FOREIGN KEY (protocol_mapper_id) REFERENCES public.protocol_mapper(id);


--
-- Name: default_client_scope fk_r_def_cli_scope_realm; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.default_client_scope
    ADD CONSTRAINT fk_r_def_cli_scope_realm FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: required_action_provider fk_req_act_realm; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.required_action_provider
    ADD CONSTRAINT fk_req_act_realm FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: resource_uris fk_resource_server_uris; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.resource_uris
    ADD CONSTRAINT fk_resource_server_uris FOREIGN KEY (resource_id) REFERENCES public.resource_server_resource(id);


--
-- Name: role_attribute fk_role_attribute_id; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.role_attribute
    ADD CONSTRAINT fk_role_attribute_id FOREIGN KEY (role_id) REFERENCES public.keycloak_role(id);


--
-- Name: realm_supported_locales fk_supported_locales_realm; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.realm_supported_locales
    ADD CONSTRAINT fk_supported_locales_realm FOREIGN KEY (realm_id) REFERENCES public.realm(id);


--
-- Name: user_federation_config fk_t13hpu1j94r2ebpekr39x5eu5; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_federation_config
    ADD CONSTRAINT fk_t13hpu1j94r2ebpekr39x5eu5 FOREIGN KEY (user_federation_provider_id) REFERENCES public.user_federation_provider(id);


--
-- Name: user_group_membership fk_user_group_user; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.user_group_membership
    ADD CONSTRAINT fk_user_group_user FOREIGN KEY (user_id) REFERENCES public.user_entity(id);


--
-- Name: policy_config fkdc34197cf864c4e43; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.policy_config
    ADD CONSTRAINT fkdc34197cf864c4e43 FOREIGN KEY (policy_id) REFERENCES public.resource_server_policy(id);


--
-- Name: identity_provider_config fkdc4897cf864c4e43; Type: FK CONSTRAINT; Schema: public; Owner: keycloak
--

ALTER TABLE ONLY public.identity_provider_config
    ADD CONSTRAINT fkdc4897cf864c4e43 FOREIGN KEY (identity_provider_id) REFERENCES public.identity_provider(internal_id);


--
-- PostgreSQL database dump complete
--

\unrestrict 20yC9tXS2fLtRIfQaX1XoyQvrCv3DQIgGdME3VPjlZdRoy42LNnXP26K2ctb9Cy

