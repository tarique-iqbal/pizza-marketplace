CREATE TABLE geocode (
    address_hash CHAR(64),
    lat DOUBLE PRECISION NOT NULL
        CONSTRAINT ck_geocode_lat
        CHECK (lat BETWEEN -90 AND 90),
    lon DOUBLE PRECISION NOT NULL
        CONSTRAINT ck_geocode_lon
        CHECK (lon BETWEEN -180 AND 180),
    created_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT pk_geocode
        PRIMARY KEY (address_hash)
);
