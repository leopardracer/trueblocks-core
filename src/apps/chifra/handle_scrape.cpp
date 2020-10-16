/*-------------------------------------------------------------------------
 * This source code is confidential proprietary information which is
 * copyright (c) 2018, 2019 TrueBlocks, LLC (http://trueblocks.io)
 * All Rights Reserved
 *------------------------------------------------------------------------*/
#include "options.h"

extern bool visitMonitor(const string_q& path, void* data);
extern bool isScraperRunning(const string_q& unsearch);
extern bool freshen_internal_for_scrape(freshen_e mode, CMonitorArray& fa, const string_q& tool_flags,
                                        const string_q& freshen_flags);
//------------------------------------------------------------------------------------------------
bool COptions::handle_scrape(void) {
    ENTER("handle_" + mode);

    // scrape mode requires a running node
    nodeRequired();

    if (contains(tool_flags, "help")) {
        ostringstream os;
        os << "blockScrape --help";
        LOG_CALL(os.str());
        // clang-format off
        if (system(os.str().c_str())) {}  // Don't remove cruft. Silences compiler warnings
        // clang-format on
        EXIT_NOMSG(true);
    }

    // syntactic sugar
    tool_flags = substitute(substitute(" " + tool_flags, "--start", " start"), " start", "");
    bool daemonMode = false;

    // The presence of 'waitFile' will either pause or kill the scraper. If 'waitFile'
    // disappears and the scraper is paused, the scraper will resume
    string_q waitFile = configPath("cache/tmp/scraper-off.txt");
    bool wasPaused = fileExists(waitFile);
    if (wasPaused)
        LOG_INFO("The scraper is currently paused");

    if (contains(tool_flags, "restart")) {
        //---------------------------------------------------------------------------------
        // If a seperate instance is not running, we can't restart it
        if (!isScraperRunning("restart")) {
            LOG_WARN("Scraper is not running. Cannot restart...");
            EXIT_NOMSG(true);
        }

        // If it's not paused, let the user know...
        if (!wasPaused) {
            LOG_WARN("Scraper is not paused. Cannot restart...");
            EXIT_NOMSG(true);
        }

        // A seperate instance is running, removing the file will restart the scraper
        ::remove(waitFile.c_str());
        LOG_INFO("Scraper will restart shortly...");
        EXIT_NOMSG(true);

    } else if (contains(tool_flags, "pause")) {
        //---------------------------------------------------------------------------------
        // If a seperate instance is not running, we can't pause it
        if (!isScraperRunning("pause")) {
            LOG_WARN("Scraper is not running. Cannot pause...");
            EXIT_NOMSG(true);
        }

        // If it's already paused, let the user know...
        if (wasPaused) {
            LOG_WARN("Scraper is already paused...");
            EXIT_NOMSG(true);
        }

        // It's running, so we can pause it
        stringToAsciiFile(waitFile, Now().Format(FMT_EXPORT));
        LOG_INFO("Scraper will pause shortly...");
        EXIT_NOMSG(true);

    } else if (contains(tool_flags, "quit")) {
        if (!isScraperRunning("quit")) {
            LOG_WARN("Scraper is not running. Cannot quit...");
            EXIT_NOMSG(true);
        }

        // Kill it whether it's currently paused or not
        stringToAsciiFile(waitFile, "quit");
        LOG_INFO("Scraper will quit shortly...");
        EXIT_NOMSG(true);

    } else {
        //---------------------------------------------------------------------------------
        // If it's already running, don't start it again...
        if (isScraperRunning("unused")) {
            LOG_WARN("Scraper is already running. Cannot start it again...");
            EXIT_NOMSG(false);
        }

        // Extract options from the command line that we do not pass on to blockScrape...
        CStringArray optList;
        explode(optList, tool_flags, ' ');
        tool_flags = "";  // reset tool_flag
        for (auto opt : optList) {
            if (opt == "--daemon") {
                opt = "";
                daemonMode = true;

            } else if (!opt.empty() && !startsWith(opt, "-") && !isNumeral(opt)) {
                cerr << "Invalid command " << cYellow << opt << cOff << " to chifra scrape." << endl;
                EXIT_NOMSG(false);
            }
            if (!opt.empty())
                tool_flags += (opt + " ");
        }

        cerr << cYellow << "Scraper is starting with " << tool_flags << "..." << cOff << endl;
    }

    timestamp_t userSleep = (scrapeSleep == 0 ? 500000 : scrapeSleep * 1000000);

    // Run forever...unless told to pause or stop (shouldQuit is true if control+C was hit.
    bool waitFileExists = fileExists(waitFile);
    size_t nRuns = 0;
    size_t maxRuns = (isTestMode() ? 1 : UINT64_MAX);
    while (nRuns++ < maxRuns && !shouldQuit()) {
        if (waitFileExists) {
            if (!wasPaused)
                cerr << cYellow << "\tScraper paused..." << cOff << endl;
            wasPaused = true;
            usleep(max(useconds_t(5), scrapeSleep) * 1000000);  // sleep for at least five seconds

        } else {
            timestamp_t startTs = date_2_Ts(Now());
            if (wasPaused)
                cerr << cYellow << "\tScraper restarted..." << cOff << endl;
            wasPaused = false;
            ostringstream os;
            os << "blockScrape " << tool_flags;
            LOG_CALL(os.str());
            // clang-format off
            if (system(os.str().c_str())) {}  // Don't remove cruft. Silences compiler warnings
            // clang-format on

            if (isTestMode()) {
                // Do nothing related in --daemon mode while testing

            } else {
                CMonitorArray monitors;
                if (daemonMode) {
                    // Catch the monitors addresses up to the scraper if in --deamon mode
                    forEveryFileInFolder(getMonitorPath("") + "*", visitMonitor, &monitors);

                    freshen_internal_for_scrape(FM_PRODUCTION, monitors, "", freshen_flags);
                    for (auto monitor : monitors) {
                        if (true) { //monitor.needsRefresh) {
                            static size_t nThings = 0;
                            ostringstream os1;
                            os1 << "acctExport " << monitor.address << " --freshen";  // << " >/dev/null";
                            LOG_INFO("Calling: ", os1.str(), string_q(40, ' '), "\r");
                            // clang-format off
                            if (system(os1.str().c_str())) {}  // Don't remove cruft. Silences compiler warnings
                            // clang-format on
                            if (shouldQuit())
                                continue;
                            if (!(nThings % 10))
                                usleep(125000);  // stay responsive to cntrl+C
                        }
                    }
                }
                timestamp_t now = max(startTs, date_2_Ts(Now())); // not less than
                timestamp_t timeSpent = (now - startTs) * 1000000;
                timestamp_t sleepSecs = timeSpent > userSleep ? 0 : userSleep - timeSpent;
                //LOG_INFO("startTs: ", startTs, " now: ", now, " timeSpent: ", timeSpent, " userSleep: ", userSleep, " sleepSecs: ", sleepSecs, string_q(60, ' '));
                if (daemonMode)
                    LOG_INFO(cYellow, "Finished freshening ", monitors.size(), " monitored addresses. Sleeping for ",
                             (sleepSecs / 1000000), " seconds", string_q(40, ' '), cOff);
                usleep(useconds_t(sleepSecs));  // stay responsive to cntrl+C
            }
        }

        waitFileExists = fileExists(waitFile);
        if (waitFileExists) {
            if (asciiFileToString(waitFile) == "quit") {
                ::remove(waitFile.c_str());
                cerr << cYellow << "\tScraper quitting..." << cOff << endl;
                EXIT_NOMSG(true);
            }
        }
    }
    EXIT_NOMSG(true);
}

//---------------------------------------------------------------------------
bool visitMonitor(const string_q& path, void* data) {
    if (!endsWith(path, ".acct.bin"))  // we only want to process monitor files
        return true;

    CMonitor m;
    m.address = substitute(substitute(path, getMonitorPath(""), ""), ".acct.bin", "");
    if (isAddress(m.address)) {
        m.cntBefore = m.getRecordCount();
        m.needsRefresh = false;
        CMonitorArray* array = (CMonitorArray*)data;  // NOLINT
        array->push_back(m);
//        LOG_INFO(cTeal, "Loading addresses ", m.address, " ", array->size(), string_q(80, ' '), cOff, "\r");
    }

    return true;
}

//----------------------------------------------------------------------------
bool isScraperRunning(const string_q& unsearch) {
    string_q pList = listProcesses("chifra scrape");
    replace(pList, "  ", " ");
    replace(pList, "chifra scrape " + unsearch, "");
    return contains(pList, "chifra scrape");
}

//------------------------------------------------------------------------------------------------
blknum_t lastExported(const address_t& addr) {
    string_q filename = getMonitorExpt(addr, FM_PRODUCTION);
    blknum_t second;
    timestamp_t unused;
    bnFromPath(getMonitorExpt(addr, FM_PRODUCTION), second, unused);
    return second == NOPOS ? 0 : second;
}

//------------------------------------------------------------------------------------------------
bool freshen_internal_for_scrape(freshen_e mode, CMonitorArray& fa, const string_q& tool_flags,
                                 const string_q& freshen_flags) {
    ENTER("freshen_internal_for_scrape");

    ostringstream base;
    base << "acctScrape " << tool_flags << " " << freshen_flags << " [ADDRS] ;";

    //blknum_t latestCache = getLatestBlock_cache_final();
    size_t cnt = 0, cnt2 = 0;
    string_q tenAddresses;
    for (auto f : fa) {
        bool needsUpdate = false; //true;
//        if (!contains(freshen_flags, "staging") && !contains(freshen_flags, "unripe")) {
//            needsUpdate = lastExported(f.address) < latestCache;
//            LOG_INFO("lastExport: ", lastExported(f.address), " latestCache: ", latestCache, " needsUpdate: ", needsUpdate);
////            getchar();
//        }
        if (needsUpdate) {
            LOG_INFO(cTeal, "Needs update ", f.address, string_q(80, ' '), cOff);
            tenAddresses += (f.address + " ");
            if (!(++cnt % 10)) {  // we don't want to do too many addrs at a time
                tenAddresses += "|";
                cnt = 0;
            }
        } else {
            LOG_INFO(cTeal, "Scraping addresses ", f.address, " ", cnt2, " of ", fa.size(), string_q(80, ' '), cOff, "\r");
        }
        cnt2++;
    }

    // Process them until we're done
    uint64_t cur = 0;
    while (!tenAddresses.empty()) {
        string_q thisFive = nextTokenClear(tenAddresses, '|');
        string_q cmd = substitute(base.str(), "[ADDRS]", thisFive);
        LOG_CALL(cmd);
        // clang-format off
        uint64_t n = countOf(thisFive, ' ');
        if (fa.size() > 1)
            LOG_INFO(cTeal, "Scraping addresses ", cur, "-", (cur+n-1), " of ", fa.size(), string_q(80, ' '), cOff);
        cur += n;
        // Don't remove cruft. Silences compiler warnings
        if (system(cmd.c_str())) {}  // clang-format on
        if (!tenAddresses.empty())
            usleep(250000);  // this sleep is here so that chifra remains responsive to Cntl+C. Do not remove
    }

    for (CMonitor& f : fa)
        f.needsRefresh = (f.cntBefore != f.getRecordCount());

    EXIT_NOMSG(true);
}
