// Copyright 2020 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy
// of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

import React from 'react'
import { useHistory, useRouteMatch } from 'react-router-dom'

import { Box, Button, Divider, IconButton, makeStyles, Menu, MenuItem, Tooltip } from '@material-ui/core'

import { MuiMarkdown, MuiMarkdownProps } from 'components/muimarkdown'
import { ChapterSkeleton } from 'components/muimarkdown/ChapterSkeleton'
import { ChevronLeft, ChevronRight, Toc as TocIcon } from '@material-ui/icons'

const navigatorBorder = 1 // px
const navigatorLeftPadding = 4 // px
const navigatorFooterSpacing = 3 // spacing(x)

const useStyles = makeStyles((theme) => ({
    canvas: {
        overflow: 'auto', // let there be shcrollbarrs!
    },
    navigator: {
        zIndex: 2,
        position: 'sticky', // within the .view, not the viewport :)
        top: theme.spacing(2),
        left: 0,
        // make the touch ripple fit in snuggly; we need to keep enough height
        // for everything, including the border (times 2) and icons are:
        // - 24px high
        // - small buttons have a 3px padding (*2)
        height: `calc(24px + ${(navigatorBorder + 2) * 2}px + 2 * 3px)`,
        // icons are:
        // - 24px wide, 
        // - small icon buttons have a 3px padding (*2),
        // - 1px border (*1, only "left side"!)
        marginLeft: `calc(100% - 24px - 6px - ${navigatorBorder}px - ${navigatorLeftPadding}px)`,
        paddingLeft: `${navigatorLeftPadding + 3}px`,
        background: theme.palette.background.paper,
        border: `${navigatorBorder}px solid ${theme.palette.type === 'light' ? 'rgba(0, 0, 0, 0.23)' : 'rgba(255, 255, 255, 0.23)'}`,
        borderRight: 0,
        borderRadius: '42em',
        borderTopRightRadius: 0,
        borderBottomRightRadius: 0,
        '& .MuiSvgIcon-root': { position: 'relative', left: '-2px' },
        '& .MuiTouchRipple-root': { left: '-1px' },

        // Material UI's icon button on hover slightly darkens the background
        // using an alpha of 0.04; now, that would make any text under the toc
        // button suddenly shine through ... and we don't want that. So we need
        // to set a non-transparent color which is appropriately darkened (or
        // lightened, depending on theme type).
        '&:hover': {
            backgroundColor: theme.palette.type === 'light' ? 'rgb(245, 245, 245)' : 'rgb(10, 10, 10)',
        },
    },
    padding: {
        marginLeft: theme.spacing(2),
        marginRight: theme.spacing(2),

        '& > hr': {
            marginTop: theme.spacing(navigatorFooterSpacing),
            marginBottom: theme.spacing(navigatorFooterSpacing) - 6,
        },
        '& > button.prev': {
            float: 'left',
        },
        '& > button.next': {
            float: 'right',
        },
        '& > button.prev, & > button.next': {
            marginBottom: theme.spacing(navigatorFooterSpacing) - 6,
        },
    },
    markdown: {
        // Compensate for the height of the sticky toc navigator button.
        marginTop: '-24px',
    },
    fallback: {
        // Compensate for the height of the sticky toc navigator button.
        marginTop: '-24px',
    }
}))

/**
 * A single help chapter.
 */
export interface HelpViewerChapter {
    /** chapter title to show in ToC and bottom navigation. */
    title: string
    /** the help chapter contents. */
    chapter: (props: any) => JSX.Element
    /** 
     * optional chapter slug, relative to base of help viewer path; if left
     * undefined, then defaults to the "slugified" chapter title, where the
     * title is converted to all lowercase, spaces and any characters outside
     * the range of 0x20-0x73 are removed completely.
     */
    slug?: string
}

/**
 * Returns either the explicitly specified chapter slug or a slug automatically
 * derived from the chapter title.
 *
 * @param chapter chapter object with title and optional slug.
 */
const slugify = (chapter: HelpViewerChapter) => (
    chapter.slug
    || chapter.title.toLowerCase().replace(/\s+/g, '').replace(/[^\x20-\x7e]/g, '')
)

/**
 * Returns the index of the chapter identified by a particular slug, or 0
 * (=first chapter) if no match could be found.
 *
 * @param slug chapter slug to find corresponding chapter for.
 * @param chapters list of chapters.
 */
const findChapter = (slug: string, chapters: HelpViewerChapter[]) => {
    const chidx = chapters.findIndex((chapter) => slug === slugify(chapter))
    return chidx >= 0 ? chidx : 0
}


export interface HelpViewerProps {
    /** 
     * list of chapters, with title and chapter fields, and an optional slug
     * field when the route slug is to be controlled explicitly (instead of
     * using an autogenerated slug) 
     */
    chapters: HelpViewerChapter[]
    /** 
     * base route path (such as "/help", et cetera), defaults to "/" if left
     * undefined.
     */
    baseroute?: string
    /**
     * The markdown renderer component type to use; defaults to
     * [MuiMarkdown](#MuiMarkdown). And yes, I've worked for too long with Go
     * interfaces...
     */
    markdowner?: (props: MuiMarkdownProps) => JSX.Element
    /** shortcodes, that is, available components. */
    shortcodes?: { [key: string]: React.ComponentType<any> }
    /** inline styles. */
    style?: React.CSSProperties
}

/**
 * A multi-page help view component including "chapter" navigation. The rendered
 * chapter (in MDX) is selected via the current route. When selecting a
 * different chapter, the component will change the route in order to show it.
 *
 * Chapter navigation:
 *
 * - previous/next chapter buttons at the end of each chapter.
 * - ToC navigation button which pops up a ToC menu.
 *
 * This help component defaults to using the [MuiMarkdown](#muimarkdown) MDX
 * renderer, which uses Material-UI typography.
 *
 * > **Important:** the ToC navigation button is sticky. Now for it to correctly
 * > stick in place as percepted by users as opposed to what the DOM does, the
 * > `HelpViewer` component **must not** be placed in an outer element which
 * > somehow handles overflows, such as showing scroll bars. Instead, the outer
 * > element **must have a fixed size** and stick to that. It's only the inner
 * > area of the help viewer that is allowed to overflow and thus shows scroll
 * > bars. And it's not possible to use "position: absolute" as this would
 * > position the ToC button absolute with respect to the complete viewport, but
 * > not the component we've stuck into... 🥴
 * >
 * > See also [CSS Position Sticky – How It Really
 * > Works!](https://medium.com/@elad/css-position-sticky-how-it-really-works-54cd01dc2d46)
 * > for some helpful insights. It actually all does make sense, but you need to
 * > dive into it.
 */
export const HelpViewer = ({ chapters, baseroute, markdowner, shortcodes, style }: HelpViewerProps) => {

    const classes = useStyles()

    // for using the component constructure in JSX versus component instances,
    // please see: https://stackoverflow.com/a/31815634/
    const MDX = markdowner || MuiMarkdown

    const m = useRouteMatch((baseroute || '') + '/:chapter')
    const currentChapterIndex = (m && m.params['chapter'] && findChapter(m.params['chapter'], chapters)) || 0

    // We need to change history when navigating to a new chapter ;)
    const history = useHistory()

    // Renders a chapter button linking to a specific chapter, or nothing if the
    // chapter index is out of range. Changes the route when clicked (taking the
    // base into account).
    const ChapterButton = ({ chapterIndex }) => {
        if (chapterIndex < 0 || chapterIndex >= chapters.length) {
            return null
        }

        const next = chapterIndex > currentChapterIndex

        return (<Button
            className={next ? 'next' : 'prev'}
            startIcon={!next && <ChevronLeft />}
            endIcon={next && <ChevronRight />}
            onClick={() => history.push(`${baseroute || '/'}/${slugify(chapters[chapterIndex])}`)}
        >
            {chapters[chapterIndex].title}
        </Button>)
    }


    // Anchor state for the ToC navigation popup menu.
    const [anchorEl, setAnchorEl] = React.useState(null)

    // Pop up the table of contents menu...
    const handleIconClick = (event) => {
        setAnchorEl(event.currentTarget)
    }

    // close popup menu, change route...
    const handleMenuItemClick = (event, index) => {
        history.push(`${baseroute || '/'}/${slugify(chapters[index])}`)
        setAnchorEl(null);
    }

    // just close that popup menu!
    const handleClose = () => {
        setAnchorEl(null);
    }

    return <div className={classes.canvas} style={style}>
        {/* 
            the ToC navigation/navigator button is sticky, but of course in the
            *outer* "canvas" div. In consequence, the outer canvas must not
            overflow! Instead, the inner markdown area must overflow and scroll.
        */}
        <Tooltip title="open table of contents">
            <IconButton
                className={classes.navigator}
                size="small"
                onClick={handleIconClick}
            >
                <TocIcon />
            </IconButton>
        </Tooltip>
        <Menu
            id="help-viewer-menu"
            anchorEl={anchorEl}
            keepMounted
            open={Boolean(anchorEl)}
            onClose={handleClose}
        >
            {chapters.map((chapter, index) => (
                <MenuItem
                    key={index}
                    selected={index === currentChapterIndex}
                    onClick={(event) => handleMenuItemClick(event, index)}
                >
                    {chapter.title}
                </MenuItem>
            ))}
        </Menu>
        <div className={classes.padding}>
            <MDX
                className={classes.markdown}
                mdx={chapters[currentChapterIndex].chapter}
                fallback={
                    <Box className={classes.fallback} m={1}>
                        <ChapterSkeleton />
                    </Box>
                }
                shortcodes={shortcodes}
            />
            <Divider />
            <ChapterButton chapterIndex={currentChapterIndex - 1} />
            <ChapterButton chapterIndex={currentChapterIndex + 1} />
            <div style={{clear: 'both'}}></div>
        </div>
    </div>
}

export default HelpViewer
