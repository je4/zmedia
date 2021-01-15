--
-- PostgreSQL database dump
--

-- Dumped from database version 11.5 (Ubuntu 11.5-1.pgdg18.04+1)
-- Dumped by pg_dump version 13.1

-- Started on 2021-01-15 10:02:56

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

--
-- TOC entry 3 (class 2615 OID 2200)
-- Name: public; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA public;


ALTER SCHEMA public OWNER TO postgres;

--
-- TOC entry 3111 (class 0 OID 0)
-- Dependencies: 3
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: postgres
--

COMMENT ON SCHEMA public IS 'standard public schema';


--
-- TOC entry 618 (class 1247 OID 16709)
-- Name: generic_sort_direction; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.generic_sort_direction AS ENUM (
    'asc',
    'desc'
);


ALTER TYPE public.generic_sort_direction OWNER TO postgres;

--
-- TOC entry 621 (class 1247 OID 16714)
-- Name: master_objecttype; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.master_objecttype AS ENUM (
    'file',
    'dir',
    'other'
);


ALTER TYPE public.master_objecttype OWNER TO postgres;

--
-- TOC entry 624 (class 1247 OID 16722)
-- Name: master_status; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.master_status AS ENUM (
    'index',
    'object',
    'other'
);


ALTER TYPE public.master_status OWNER TO postgres;

--
-- TOC entry 627 (class 1247 OID 16730)
-- Name: master_type; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.master_type AS ENUM (
    'image',
    'audio',
    'video',
    'waypoint',
    '3d',
    'document',
    'office',
    'pdf',
    'cdn',
    'default',
    'none',
    'gpx'
);


ALTER TYPE public.master_type OWNER TO postgres;

--
-- TOC entry 630 (class 1247 OID 16756)
-- Name: object_statustype; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.object_statustype AS ENUM (
    'new',
    'inprocess',
    'deleted',
    'approved'
);


ALTER TYPE public.object_statustype OWNER TO postgres;

--
-- TOC entry 633 (class 1247 OID 16767)
-- Name: sort_direction; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.sort_direction AS (
	"asc" "char",
	"desc" "char"
);


ALTER TYPE public.sort_direction OWNER TO postgres;

--
-- TOC entry 233 (class 1255 OID 16768)
-- Name: createobject(character varying, character varying, text, jsonb, character varying); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.createobject(_objecttypename character varying, _title character varying, _fulltext text, _data jsonb, _creator character varying) RETURNS character varying
    LANGUAGE plpgsql
    AS $$
DECLARE 
	otypeid bigint;
	oid bigint;
BEGIN
	otypeid = (SELECT t.objecttypeid from objecttype t where t.name = _objecttypename);
	
	INSERT INTO "object" ( objecttypeid,
						   status,
						  title,
						  fulltext,
						  "data",
						  creator,
						  modifier) 
	VALUES (otypeid, 'new', _title, _fulltext, _data, _creator, _creator )
	RETURNING objectid INTO oid;
	RETURN format('%s-%s', _objecttypename, oid::character varying);
END;
$$;


ALTER FUNCTION public.createobject(_objecttypename character varying, _title character varying, _fulltext text, _data jsonb, _creator character varying) OWNER TO postgres;

--
-- TOC entry 234 (class 1255 OID 16769)
-- Name: findobject(text, text, public.generic_sort_direction); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.findobject(t text, sortcol text, sortdesc public.generic_sort_direction) RETURNS TABLE(objecttype character varying, objectid bigint, status public.object_statustype, title character varying, data jsonb, modified timestamp without time zone, modifier character varying)
    LANGUAGE plpgsql
    AS $$DECLARE
BEGIN
	RETURN QUERY EXECUTE format( 'SELECT ot.name, 
		o.objectid, 
		o.status, 
		o.title, 
		o.data, 
		o.modified, 
		o.modifier 
	FROM public.object o, 
		public.objecttype ot 
	WHERE o.objecttypeid = ot.objecttypeid 
		AND ot.name = %L
	ORDER BY %I %s', t, sortcol, sortdesc );
END
$$;


ALTER FUNCTION public.findobject(t text, sortcol text, sortdesc public.generic_sort_direction) OWNER TO postgres;

--
-- TOC entry 235 (class 1255 OID 16770)
-- Name: findobject(text, text, text, public.generic_sort_direction); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.findobject(t text, _title text, sortcol text, sortdesc public.generic_sort_direction) RETURNS TABLE(objecttype character varying, objectid bigint, status public.object_statustype, title character varying, data jsonb, modified timestamp without time zone, modifier character varying)
    LANGUAGE plpgsql
    AS $$
DECLARE
BEGIN
	RETURN QUERY EXECUTE format( 'SELECT ot.name, 
		o.objectid, 
		o.status, 
		o.title, 
		o.data, 
		o.modified, 
		o.modifier 
	FROM public.object o, 
		public.objecttype ot 
	WHERE o.objecttypeid = ot.objecttypeid
		  AND ot.name = %L 
		  AND title LIKE %L
	ORDER BY %I %s', t, _title, sortcol, sortdesc );
END
$$;


ALTER FUNCTION public.findobject(t text, _title text, sortcol text, sortdesc public.generic_sort_direction) OWNER TO postgres;

--
-- TOC entry 236 (class 1255 OID 16771)
-- Name: findobject(text, tsquery, text, public.generic_sort_direction); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.findobject(t text, _ft tsquery, sortcol text, sortdesc public.generic_sort_direction DEFAULT 'asc'::public.generic_sort_direction) RETURNS TABLE(objecttype character varying, objectid bigint, status public.object_statustype, title character varying, data jsonb, modified timestamp without time zone, modifier character varying)
    LANGUAGE plpgsql
    AS $$
DECLARE
BEGIN
	RETURN QUERY EXECUTE format( 'SELECT ot.name, 
		o.objectid, 
		o.status, 
		o.title, 
		o.data, 
		o.modified, 
		o.modifier 
	FROM public.object o, 
		public.objecttype ot 
	WHERE o.objecttypeid = ot.objecttypeid
		  AND ot.name = %L 
		  AND fulltext @@ %L::tsquery
	ORDER BY %I %s', t, _ft, sortcol, sortdesc );
END
$$;


ALTER FUNCTION public.findobject(t text, _ft tsquery, sortcol text, sortdesc public.generic_sort_direction) OWNER TO postgres;

--
-- TOC entry 237 (class 1255 OID 16772)
-- Name: findobject(text, text, tsquery, text, public.generic_sort_direction); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.findobject(t text, _title text, _ft tsquery, sortcol text, sortdesc public.generic_sort_direction) RETURNS TABLE(objecttype character varying, objectid bigint, status public.object_statustype, title character varying, data jsonb, modified timestamp without time zone, modifier character varying)
    LANGUAGE plpgsql
    AS $$
DECLARE
BEGIN
	RETURN QUERY EXECUTE format( 'SELECT ot.name, 
		o.objectid, 
		o.status, 
		o.title, 
		o.data, 
		o.modified, 
		o.modifier 
	FROM public.object o, 
		public.objecttype ot 
	WHERE o.objecttypeid = ot.objecttypeid
		  AND ot.name = %L 
		  AND title LIKE %L
		  AND fulltext @@ %L::tsquery
	ORDER BY %I %s', t, _title, _ft, sortcol, sortdesc );
END
$$;


ALTER FUNCTION public.findobject(t text, _title text, _ft tsquery, sortcol text, sortdesc public.generic_sort_direction) OWNER TO postgres;

SET default_tablespace = '';

--
-- TOC entry 198 (class 1259 OID 16773)
-- Name: cache; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.cache (
    cacheid bigint NOT NULL,
    masterid bigint NOT NULL,
    storageid bigint NOT NULL,
    action text NOT NULL,
    param text,
    width bigint,
    height bigint,
    duration bigint DEFAULT '0'::bigint,
    mimetype text,
    filesize bigint NOT NULL,
    path text,
    cachetime timestamp with time zone DEFAULT now() NOT NULL,
    lastaccess timestamp with time zone DEFAULT now() NOT NULL,
    collectionid bigint NOT NULL
);


ALTER TABLE public.cache OWNER TO postgres;

--
-- TOC entry 199 (class 1259 OID 16782)
-- Name: cache_cacheid_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.cache_cacheid_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.cache_cacheid_seq OWNER TO postgres;

--
-- TOC entry 3118 (class 0 OID 0)
-- Dependencies: 199
-- Name: cache_cacheid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.cache_cacheid_seq OWNED BY public.cache.cacheid;


--
-- TOC entry 200 (class 1259 OID 16784)
-- Name: cache_data; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.cache_data (
    cacheid bigint NOT NULL,
    data jsonb
);


ALTER TABLE public.cache_data OWNER TO postgres;

--
-- TOC entry 201 (class 1259 OID 16790)
-- Name: collection; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.collection (
    collectionid bigint NOT NULL,
    estateid bigint DEFAULT '1'::numeric NOT NULL,
    name text NOT NULL,
    description text,
    signature_prefix text,
    storageid bigint NOT NULL,
    json text,
    zoterogroup bigint
);


ALTER TABLE public.collection OWNER TO postgres;

--
-- TOC entry 202 (class 1259 OID 16797)
-- Name: collection_collectionid_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.collection_collectionid_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.collection_collectionid_seq OWNER TO postgres;

--
-- TOC entry 3122 (class 0 OID 0)
-- Dependencies: 202
-- Name: collection_collectionid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.collection_collectionid_seq OWNED BY public.collection.collectionid;


--
-- TOC entry 203 (class 1259 OID 16799)
-- Name: estate; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.estate (
    estateid bigint NOT NULL,
    name text NOT NULL,
    description text
);


ALTER TABLE public.estate OWNER TO postgres;

--
-- TOC entry 204 (class 1259 OID 16805)
-- Name: estate_estateid_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.estate_estateid_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.estate_estateid_seq OWNER TO postgres;

--
-- TOC entry 3125 (class 0 OID 0)
-- Dependencies: 204
-- Name: estate_estateid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.estate_estateid_seq OWNED BY public.estate.estateid;


--
-- TOC entry 205 (class 1259 OID 16807)
-- Name: master; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.master (
    masterid bigint NOT NULL,
    collectionid bigint NOT NULL,
    signature text NOT NULL,
    urn text NOT NULL,
    type public.master_type,
    subtype text,
    objecttype public.master_objecttype DEFAULT 'file'::public.master_objecttype NOT NULL,
    status public.master_status DEFAULT 'other'::public.master_status NOT NULL,
    parentid bigint,
    mimetype text,
    error text,
    sha256 character(65),
    metadata jsonb
);


ALTER TABLE public.master OWNER TO postgres;

--
-- TOC entry 206 (class 1259 OID 16815)
-- Name: master_masterid_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.master_masterid_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.master_masterid_seq OWNER TO postgres;

--
-- TOC entry 3128 (class 0 OID 0)
-- Dependencies: 206
-- Name: master_masterid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.master_masterid_seq OWNED BY public.master.masterid;


--
-- TOC entry 207 (class 1259 OID 16817)
-- Name: object; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.object (
    objectid bigint NOT NULL,
    objecttypeid bigint NOT NULL,
    status public.object_statustype DEFAULT 'new'::public.object_statustype NOT NULL,
    title character varying(1024),
    fulltext text,
    data jsonb,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    creator character varying(255) NOT NULL,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    modifier character varying(255) NOT NULL,
    oldid character varying(255) DEFAULT NULL::character varying,
    deleted boolean DEFAULT false NOT NULL
);


ALTER TABLE public.object OWNER TO postgres;

--
-- TOC entry 208 (class 1259 OID 16828)
-- Name: object_objectid_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.object_objectid_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.object_objectid_seq OWNER TO postgres;

--
-- TOC entry 3131 (class 0 OID 0)
-- Dependencies: 208
-- Name: object_objectid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.object_objectid_seq OWNED BY public.object.objectid;


--
-- TOC entry 209 (class 1259 OID 16830)
-- Name: objecttype; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.objecttype (
    objecttypeid bigint NOT NULL,
    name character varying(255) NOT NULL,
    template character varying(255)
);


ALTER TABLE public.objecttype OWNER TO postgres;

--
-- TOC entry 3133 (class 0 OID 0)
-- Dependencies: 209
-- Name: TABLE objecttype; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.objecttype IS 'list of object types with templates';


--
-- TOC entry 210 (class 1259 OID 16836)
-- Name: object_type_object_type_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.object_type_object_type_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.object_type_object_type_id_seq OWNER TO postgres;

--
-- TOC entry 3135 (class 0 OID 0)
-- Dependencies: 210
-- Name: object_type_object_type_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.object_type_object_type_id_seq OWNED BY public.objecttype.objecttypeid;


--
-- TOC entry 211 (class 1259 OID 16838)
-- Name: objectgroup; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.objectgroup (
    objectgroupid bigint NOT NULL,
    reference text NOT NULL,
    creationdate timestamp with time zone DEFAULT now() NOT NULL,
    closed boolean DEFAULT false NOT NULL
);


ALTER TABLE public.objectgroup OWNER TO postgres;

--
-- TOC entry 212 (class 1259 OID 16846)
-- Name: objectgroup_master; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.objectgroup_master (
    objectgroupid bigint NOT NULL,
    masterid bigint NOT NULL
);


ALTER TABLE public.objectgroup_master OWNER TO postgres;

--
-- TOC entry 213 (class 1259 OID 16849)
-- Name: objectgroup_objectgroupid_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.objectgroup_objectgroupid_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.objectgroup_objectgroupid_seq OWNER TO postgres;

--
-- TOC entry 3139 (class 0 OID 0)
-- Dependencies: 213
-- Name: objectgroup_objectgroupid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.objectgroup_objectgroupid_seq OWNED BY public.objectgroup.objectgroupid;


--
-- TOC entry 214 (class 1259 OID 16851)
-- Name: rights; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.rights (
    masterid numeric NOT NULL,
    rightholder text,
    license text,
    restrictedlicense text,
    access text,
    reference text,
    embargo date,
    endoflife date,
    label text,
    modifier text NOT NULL,
    modificationtime timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.rights OWNER TO postgres;

--
-- TOC entry 215 (class 1259 OID 16858)
-- Name: storage; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.storage (
    storageid bigint NOT NULL,
    name text NOT NULL,
    urlbase text NOT NULL,
    filebase text NOT NULL,
    datadir text DEFAULT 'data'::text NOT NULL,
    videodir text DEFAULT 'video'::text NOT NULL,
    submasterdir text DEFAULT 'submaster'::text,
    tempdir text DEFAULT 'temp'::text NOT NULL,
    jwtkey text
);


ALTER TABLE public.storage OWNER TO postgres;

--
-- TOC entry 216 (class 1259 OID 16868)
-- Name: storage_storageid_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.storage_storageid_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.storage_storageid_seq OWNER TO postgres;

--
-- TOC entry 3143 (class 0 OID 0)
-- Dependencies: 216
-- Name: storage_storageid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.storage_storageid_seq OWNED BY public.storage.storageid;


--
-- TOC entry 2870 (class 2604 OID 16870)
-- Name: cache cacheid; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.cache ALTER COLUMN cacheid SET DEFAULT nextval('public.cache_cacheid_seq'::regclass);


--
-- TOC entry 2872 (class 2604 OID 16871)
-- Name: collection collectionid; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection ALTER COLUMN collectionid SET DEFAULT nextval('public.collection_collectionid_seq'::regclass);


--
-- TOC entry 2873 (class 2604 OID 16872)
-- Name: estate estateid; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.estate ALTER COLUMN estateid SET DEFAULT nextval('public.estate_estateid_seq'::regclass);


--
-- TOC entry 2876 (class 2604 OID 16873)
-- Name: master masterid; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.master ALTER COLUMN masterid SET DEFAULT nextval('public.master_masterid_seq'::regclass);


--
-- TOC entry 2882 (class 2604 OID 16874)
-- Name: object objectid; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.object ALTER COLUMN objectid SET DEFAULT nextval('public.object_objectid_seq'::regclass);


--
-- TOC entry 2886 (class 2604 OID 16875)
-- Name: objectgroup objectgroupid; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.objectgroup ALTER COLUMN objectgroupid SET DEFAULT nextval('public.objectgroup_objectgroupid_seq'::regclass);


--
-- TOC entry 2883 (class 2604 OID 16876)
-- Name: objecttype objecttypeid; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.objecttype ALTER COLUMN objecttypeid SET DEFAULT nextval('public.object_type_object_type_id_seq'::regclass);


--
-- TOC entry 2892 (class 2604 OID 16877)
-- Name: storage storageid; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.storage ALTER COLUMN storageid SET DEFAULT nextval('public.storage_storageid_seq'::regclass);


--
-- TOC entry 3087 (class 0 OID 16773)
-- Dependencies: 198
-- Data for Name: cache; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.cache (cacheid, masterid, storageid, action, param, width, height, duration, mimetype, filesize, path, cachetime, lastaccess, collectionid) FROM stdin;
\.


--
-- TOC entry 3089 (class 0 OID 16784)
-- Dependencies: 200
-- Data for Name: cache_data; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.cache_data (cacheid, data) FROM stdin;
\.


--
-- TOC entry 3090 (class 0 OID 16790)
-- Dependencies: 201
-- Data for Name: collection; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.collection (collectionid, estateid, name, description, signature_prefix, storageid, json, zoterogroup) FROM stdin;
10	14	test	testing 123	test-	31	\N	0
\.


--
-- TOC entry 3092 (class 0 OID 16799)
-- Dependencies: 203
-- Data for Name: estate; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.estate (estateid, name, description) FROM stdin;
14	test	lorem ipsum dolor sit amet
\.


--
-- TOC entry 3094 (class 0 OID 16807)
-- Dependencies: 205
-- Data for Name: master; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.master (masterid, collectionid, signature, urn, type, subtype, objecttype, status, parentid, mimetype, error, sha256, metadata) FROM stdin;
467	10	testing	file://test/test.png	\N	\N	file	other	\N	\N	\N	\N	\N
\.


--
-- TOC entry 3096 (class 0 OID 16817)
-- Dependencies: 207
-- Data for Name: object; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.object (objectid, objecttypeid, status, title, fulltext, data, created, creator, modified, modifier, oldid, deleted) FROM stdin;
1	1	new	testing123	testing abc	\N	2019-07-15 16:27:26.179536	je	2019-07-15 16:27:26.179536	je	\N	f
3	1	new	test 2	testing 3e45	\N	2019-07-16 15:25:34.779368	je	2019-07-16 15:25:34.779368	je	\N	f
\.


--
-- TOC entry 3100 (class 0 OID 16838)
-- Dependencies: 211
-- Data for Name: objectgroup; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.objectgroup (objectgroupid, reference, creationdate, closed) FROM stdin;
\.


--
-- TOC entry 3101 (class 0 OID 16846)
-- Dependencies: 212
-- Data for Name: objectgroup_master; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.objectgroup_master (objectgroupid, masterid) FROM stdin;
\.


--
-- TOC entry 3098 (class 0 OID 16830)
-- Dependencies: 209
-- Data for Name: objecttype; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.objecttype (objecttypeid, name, template) FROM stdin;
1	test	\N
\.


--
-- TOC entry 3103 (class 0 OID 16851)
-- Dependencies: 214
-- Data for Name: rights; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.rights (masterid, rightholder, license, restrictedlicense, access, reference, embargo, endoflife, label, modifier, modificationtime) FROM stdin;
\.


--
-- TOC entry 3104 (class 0 OID 16858)
-- Dependencies: 215
-- Data for Name: storage; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.storage (storageid, name, urlbase, filebase, datadir, videodir, submasterdir, tempdir, jwtkey) FROM stdin;
31	test		s3://hgk/media-test	data	video	submaster	temp	
\.


--
-- TOC entry 3145 (class 0 OID 0)
-- Dependencies: 199
-- Name: cache_cacheid_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.cache_cacheid_seq', 133, true);


--
-- TOC entry 3146 (class 0 OID 0)
-- Dependencies: 202
-- Name: collection_collectionid_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.collection_collectionid_seq', 10, true);


--
-- TOC entry 3147 (class 0 OID 0)
-- Dependencies: 204
-- Name: estate_estateid_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.estate_estateid_seq', 14, true);


--
-- TOC entry 3148 (class 0 OID 0)
-- Dependencies: 206
-- Name: master_masterid_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.master_masterid_seq', 468, true);


--
-- TOC entry 3149 (class 0 OID 0)
-- Dependencies: 208
-- Name: object_objectid_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.object_objectid_seq', 16, true);


--
-- TOC entry 3150 (class 0 OID 0)
-- Dependencies: 210
-- Name: object_type_object_type_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.object_type_object_type_id_seq', 1, true);


--
-- TOC entry 3151 (class 0 OID 0)
-- Dependencies: 213
-- Name: objectgroup_objectgroupid_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.objectgroup_objectgroupid_seq', 1, true);


--
-- TOC entry 3152 (class 0 OID 0)
-- Dependencies: 216
-- Name: storage_storageid_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.storage_storageid_seq', 31, true);


--
-- TOC entry 2905 (class 2606 OID 16879)
-- Name: cache_data cache_data_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.cache_data
    ADD CONSTRAINT cache_data_pkey PRIMARY KEY (cacheid);


--
-- TOC entry 2901 (class 2606 OID 16881)
-- Name: cache idx_16390_primary; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.cache
    ADD CONSTRAINT idx_16390_primary PRIMARY KEY (cacheid);


--
-- TOC entry 2910 (class 2606 OID 16883)
-- Name: collection idx_16402_primary; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection
    ADD CONSTRAINT idx_16402_primary PRIMARY KEY (collectionid);


--
-- TOC entry 2914 (class 2606 OID 16885)
-- Name: estate idx_16412_primary; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.estate
    ADD CONSTRAINT idx_16412_primary PRIMARY KEY (estateid);


--
-- TOC entry 2922 (class 2606 OID 16887)
-- Name: master idx_16463_primary; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.master
    ADD CONSTRAINT idx_16463_primary PRIMARY KEY (masterid);


--
-- TOC entry 2938 (class 2606 OID 16889)
-- Name: objectgroup idx_16474_primary; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.objectgroup
    ADD CONSTRAINT idx_16474_primary PRIMARY KEY (objectgroupid);


--
-- TOC entry 2941 (class 2606 OID 16891)
-- Name: objectgroup_master idx_16483_primary; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.objectgroup_master
    ADD CONSTRAINT idx_16483_primary PRIMARY KEY (objectgroupid, masterid);


--
-- TOC entry 2949 (class 2606 OID 16893)
-- Name: rights idx_16489_primary; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.rights
    ADD CONSTRAINT idx_16489_primary PRIMARY KEY (masterid);


--
-- TOC entry 2953 (class 2606 OID 16895)
-- Name: storage idx_16498_primary; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.storage
    ADD CONSTRAINT idx_16498_primary PRIMARY KEY (storageid);


--
-- TOC entry 2932 (class 2606 OID 16897)
-- Name: objecttype name; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.objecttype
    ADD CONSTRAINT name UNIQUE (name);


--
-- TOC entry 2930 (class 2606 OID 16899)
-- Name: object object_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.object
    ADD CONSTRAINT object_pkey PRIMARY KEY (objectid);


--
-- TOC entry 2934 (class 2606 OID 16901)
-- Name: objecttype object_type_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.objecttype
    ADD CONSTRAINT object_type_pkey PRIMARY KEY (objecttypeid);


--
-- TOC entry 2893 (class 1259 OID 1870995)
-- Name: fki_collection; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX fki_collection ON public.cache USING btree (collectionid);


--
-- TOC entry 2894 (class 1259 OID 16902)
-- Name: idx_16390_action; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16390_action ON public.cache USING btree (action);


--
-- TOC entry 2895 (class 1259 OID 16903)
-- Name: idx_16390_cachetime; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16390_cachetime ON public.cache USING btree (cachetime);


--
-- TOC entry 2896 (class 1259 OID 16904)
-- Name: idx_16390_duration; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16390_duration ON public.cache USING btree (duration);


--
-- TOC entry 2897 (class 1259 OID 16905)
-- Name: idx_16390_height; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16390_height ON public.cache USING btree (height);


--
-- TOC entry 2898 (class 1259 OID 16906)
-- Name: idx_16390_lastaccess; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16390_lastaccess ON public.cache USING btree (lastaccess);


--
-- TOC entry 2899 (class 1259 OID 16907)
-- Name: idx_16390_masterid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_16390_masterid ON public.cache USING btree (masterid, action, param);


--
-- TOC entry 2902 (class 1259 OID 16908)
-- Name: idx_16390_storageid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16390_storageid ON public.cache USING btree (storageid);


--
-- TOC entry 2903 (class 1259 OID 16909)
-- Name: idx_16390_width; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16390_width ON public.cache USING btree (width);


--
-- TOC entry 2906 (class 1259 OID 16910)
-- Name: idx_16402_collection_ibfk_2_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16402_collection_ibfk_2_idx ON public.collection USING btree (estateid);


--
-- TOC entry 2907 (class 1259 OID 16911)
-- Name: idx_16402_collection_ifbfk_2_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16402_collection_ifbfk_2_idx ON public.collection USING btree (estateid);


--
-- TOC entry 2908 (class 1259 OID 16912)
-- Name: idx_16402_name; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_16402_name ON public.collection USING btree (name);


--
-- TOC entry 2911 (class 1259 OID 16913)
-- Name: idx_16402_storageid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16402_storageid ON public.collection USING btree (storageid);


--
-- TOC entry 2912 (class 1259 OID 16914)
-- Name: idx_16412_name_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_16412_name_unique ON public.estate USING btree (name);


--
-- TOC entry 2915 (class 1259 OID 16915)
-- Name: idx_16463_collectionid_2; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_16463_collectionid_2 ON public.master USING btree (collectionid, signature);


--
-- TOC entry 2916 (class 1259 OID 16916)
-- Name: idx_16463_error; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16463_error ON public.master USING btree (error);


--
-- TOC entry 2917 (class 1259 OID 16917)
-- Name: idx_16463_metadata_ginp; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16463_metadata_ginp ON public.master USING gin (metadata jsonb_path_ops);


--
-- TOC entry 2918 (class 1259 OID 16918)
-- Name: idx_16463_mimetype; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16463_mimetype ON public.master USING btree (mimetype);


--
-- TOC entry 2919 (class 1259 OID 16919)
-- Name: idx_16463_objecttype; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16463_objecttype ON public.master USING btree (objecttype);


--
-- TOC entry 2920 (class 1259 OID 16920)
-- Name: idx_16463_parentid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16463_parentid ON public.master USING btree (parentid);


--
-- TOC entry 2923 (class 1259 OID 16921)
-- Name: idx_16463_signature; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16463_signature ON public.master USING btree (signature);


--
-- TOC entry 2924 (class 1259 OID 16922)
-- Name: idx_16463_status; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16463_status ON public.master USING btree (status);


--
-- TOC entry 2925 (class 1259 OID 16923)
-- Name: idx_16463_subtype; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16463_subtype ON public.master USING btree (subtype);


--
-- TOC entry 2926 (class 1259 OID 16924)
-- Name: idx_16463_type; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16463_type ON public.master USING btree (type);


--
-- TOC entry 2927 (class 1259 OID 16925)
-- Name: idx_16463_urn; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16463_urn ON public.master USING btree (urn);


--
-- TOC entry 2935 (class 1259 OID 16926)
-- Name: idx_16474_closed; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16474_closed ON public.objectgroup USING btree (closed);


--
-- TOC entry 2936 (class 1259 OID 16927)
-- Name: idx_16474_name; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_16474_name ON public.objectgroup USING btree (reference);


--
-- TOC entry 2939 (class 1259 OID 16928)
-- Name: idx_16483_masterild; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16483_masterild ON public.objectgroup_master USING btree (masterid);


--
-- TOC entry 2942 (class 1259 OID 16929)
-- Name: idx_16489_access; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16489_access ON public.rights USING btree (access);


--
-- TOC entry 2943 (class 1259 OID 16930)
-- Name: idx_16489_embargo; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16489_embargo ON public.rights USING btree (embargo);


--
-- TOC entry 2944 (class 1259 OID 16931)
-- Name: idx_16489_endoflife; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16489_endoflife ON public.rights USING btree (endoflife);


--
-- TOC entry 2945 (class 1259 OID 16932)
-- Name: idx_16489_license; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16489_license ON public.rights USING btree (license);


--
-- TOC entry 2946 (class 1259 OID 16933)
-- Name: idx_16489_modificationtime; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16489_modificationtime ON public.rights USING btree (modificationtime);


--
-- TOC entry 2947 (class 1259 OID 16934)
-- Name: idx_16489_modifier; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16489_modifier ON public.rights USING btree (modifier);


--
-- TOC entry 2950 (class 1259 OID 16935)
-- Name: idx_16489_reference; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16489_reference ON public.rights USING btree (reference);


--
-- TOC entry 2951 (class 1259 OID 16936)
-- Name: idx_16489_rightholder; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_16489_rightholder ON public.rights USING btree (rightholder);


--
-- TOC entry 2928 (class 1259 OID 16937)
-- Name: object_oldid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX object_oldid ON public.object USING btree (oldid);


--
-- TOC entry 2954 (class 1259 OID 1871138)
-- Name: storage_name_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX storage_name_uindex ON public.storage USING btree (name);


--
-- TOC entry 2958 (class 2606 OID 16938)
-- Name: cache_data cacheid; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.cache_data
    ADD CONSTRAINT cacheid FOREIGN KEY (cacheid) REFERENCES public.cache(cacheid) ON DELETE CASCADE;


--
-- TOC entry 2961 (class 2606 OID 16943)
-- Name: master collection; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.master
    ADD CONSTRAINT collection FOREIGN KEY (collectionid) REFERENCES public.collection(collectionid) ON DELETE RESTRICT;


--
-- TOC entry 2957 (class 2606 OID 1870990)
-- Name: cache collection; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.cache
    ADD CONSTRAINT collection FOREIGN KEY (collectionid) REFERENCES public.collection(collectionid) NOT VALID;


--
-- TOC entry 2959 (class 2606 OID 16948)
-- Name: collection estate; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection
    ADD CONSTRAINT estate FOREIGN KEY (estateid) REFERENCES public.estate(estateid) ON DELETE RESTRICT;


--
-- TOC entry 2955 (class 2606 OID 16953)
-- Name: cache master; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.cache
    ADD CONSTRAINT master FOREIGN KEY (masterid) REFERENCES public.master(masterid) ON DELETE RESTRICT;


--
-- TOC entry 2964 (class 2606 OID 16958)
-- Name: objectgroup_master master; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.objectgroup_master
    ADD CONSTRAINT master FOREIGN KEY (masterid) REFERENCES public.master(masterid) ON DELETE RESTRICT;


--
-- TOC entry 2965 (class 2606 OID 16963)
-- Name: objectgroup_master objectgroup; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.objectgroup_master
    ADD CONSTRAINT objectgroup FOREIGN KEY (objectgroupid) REFERENCES public.objectgroup(objectgroupid) ON DELETE RESTRICT;


--
-- TOC entry 2962 (class 2606 OID 16968)
-- Name: master parent; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.master
    ADD CONSTRAINT parent FOREIGN KEY (masterid) REFERENCES public.master(masterid) ON DELETE RESTRICT;


--
-- TOC entry 2956 (class 2606 OID 16973)
-- Name: cache storage; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.cache
    ADD CONSTRAINT storage FOREIGN KEY (storageid) REFERENCES public.storage(storageid) ON DELETE RESTRICT;


--
-- TOC entry 2960 (class 2606 OID 16978)
-- Name: collection storage; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.collection
    ADD CONSTRAINT storage FOREIGN KEY (storageid) REFERENCES public.storage(storageid);


--
-- TOC entry 2963 (class 2606 OID 16983)
-- Name: object type; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.object
    ADD CONSTRAINT type FOREIGN KEY (objecttypeid) REFERENCES public.objecttype(objecttypeid) ON DELETE CASCADE;


--
-- TOC entry 3112 (class 0 OID 0)
-- Dependencies: 233
-- Name: FUNCTION createobject(_objecttypename character varying, _title character varying, _fulltext text, _data jsonb, _creator character varying); Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON FUNCTION public.createobject(_objecttypename character varying, _title character varying, _fulltext text, _data jsonb, _creator character varying) TO media;


--
-- TOC entry 3113 (class 0 OID 0)
-- Dependencies: 234
-- Name: FUNCTION findobject(t text, sortcol text, sortdesc public.generic_sort_direction); Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON FUNCTION public.findobject(t text, sortcol text, sortdesc public.generic_sort_direction) TO media;


--
-- TOC entry 3114 (class 0 OID 0)
-- Dependencies: 235
-- Name: FUNCTION findobject(t text, _title text, sortcol text, sortdesc public.generic_sort_direction); Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON FUNCTION public.findobject(t text, _title text, sortcol text, sortdesc public.generic_sort_direction) TO media;


--
-- TOC entry 3115 (class 0 OID 0)
-- Dependencies: 236
-- Name: FUNCTION findobject(t text, _ft tsquery, sortcol text, sortdesc public.generic_sort_direction); Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON FUNCTION public.findobject(t text, _ft tsquery, sortcol text, sortdesc public.generic_sort_direction) TO media;


--
-- TOC entry 3116 (class 0 OID 0)
-- Dependencies: 237
-- Name: FUNCTION findobject(t text, _title text, _ft tsquery, sortcol text, sortdesc public.generic_sort_direction); Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON FUNCTION public.findobject(t text, _title text, _ft tsquery, sortcol text, sortdesc public.generic_sort_direction) TO media;


--
-- TOC entry 3117 (class 0 OID 0)
-- Dependencies: 198
-- Name: TABLE cache; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.cache TO media;


--
-- TOC entry 3119 (class 0 OID 0)
-- Dependencies: 199
-- Name: SEQUENCE cache_cacheid_seq; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON SEQUENCE public.cache_cacheid_seq TO media;


--
-- TOC entry 3120 (class 0 OID 0)
-- Dependencies: 200
-- Name: TABLE cache_data; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.cache_data TO media;


--
-- TOC entry 3121 (class 0 OID 0)
-- Dependencies: 201
-- Name: TABLE collection; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.collection TO media;


--
-- TOC entry 3123 (class 0 OID 0)
-- Dependencies: 202
-- Name: SEQUENCE collection_collectionid_seq; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON SEQUENCE public.collection_collectionid_seq TO media;


--
-- TOC entry 3124 (class 0 OID 0)
-- Dependencies: 203
-- Name: TABLE estate; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.estate TO media;


--
-- TOC entry 3126 (class 0 OID 0)
-- Dependencies: 204
-- Name: SEQUENCE estate_estateid_seq; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON SEQUENCE public.estate_estateid_seq TO media;


--
-- TOC entry 3127 (class 0 OID 0)
-- Dependencies: 205
-- Name: TABLE master; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.master TO media;


--
-- TOC entry 3129 (class 0 OID 0)
-- Dependencies: 206
-- Name: SEQUENCE master_masterid_seq; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON SEQUENCE public.master_masterid_seq TO media;


--
-- TOC entry 3130 (class 0 OID 0)
-- Dependencies: 207
-- Name: TABLE object; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.object TO media;


--
-- TOC entry 3132 (class 0 OID 0)
-- Dependencies: 208
-- Name: SEQUENCE object_objectid_seq; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON SEQUENCE public.object_objectid_seq TO media;


--
-- TOC entry 3134 (class 0 OID 0)
-- Dependencies: 209
-- Name: TABLE objecttype; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.objecttype TO media;


--
-- TOC entry 3136 (class 0 OID 0)
-- Dependencies: 210
-- Name: SEQUENCE object_type_object_type_id_seq; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON SEQUENCE public.object_type_object_type_id_seq TO media;


--
-- TOC entry 3137 (class 0 OID 0)
-- Dependencies: 211
-- Name: TABLE objectgroup; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.objectgroup TO media;


--
-- TOC entry 3138 (class 0 OID 0)
-- Dependencies: 212
-- Name: TABLE objectgroup_master; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.objectgroup_master TO media;


--
-- TOC entry 3140 (class 0 OID 0)
-- Dependencies: 213
-- Name: SEQUENCE objectgroup_objectgroupid_seq; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON SEQUENCE public.objectgroup_objectgroupid_seq TO media;


--
-- TOC entry 3141 (class 0 OID 0)
-- Dependencies: 214
-- Name: TABLE rights; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.rights TO media;


--
-- TOC entry 3142 (class 0 OID 0)
-- Dependencies: 215
-- Name: TABLE storage; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.storage TO media;


--
-- TOC entry 3144 (class 0 OID 0)
-- Dependencies: 216
-- Name: SEQUENCE storage_storageid_seq; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON SEQUENCE public.storage_storageid_seq TO media;


-- Completed on 2021-01-15 10:03:09

--
-- PostgreSQL database dump complete
--

