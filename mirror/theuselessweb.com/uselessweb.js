function uselessWebButton(button) {
  var initialClick = false
  var randomRange = 6

  let sitelistName = 'sitelist-mar-19'

  var sitesList = [
    'https://sliding.toys/mystic-square/8-puzzle/daily/',
    'https://longdogechallenge.com/',
    'https://maze.toys/mazes/mini/daily/',
    'https://optical.toys',
    'https://paint.toys/',
    'https://puginarug.com',
    'https://alwaysjudgeabookbyitscover.com',
    'https://clicking.toys/flip-grid/neat-nine/3-holes/',
    'https://weirdorconfusing.com/',
    'https://checkbox.toys/scale/',
    'https://memory.toys/classic/easy/',
    'https://binarypiano.com/',
    'https://mondrianandme.com/',
    'https://onesquareminesweeper.com/',
    'https://cursoreffects.com',
    'http://floatingqrcode.com/',
    'https://thatsthefinger.com/',
    'https://cant-not-tweet-this.com/',
    'http://heeeeeeeey.com/',
    'http://corndog.io/',
    'http://eelslap.com/',
    'http://www.staggeringbeauty.com/',
    'http://burymewithmymoney.com/',
    'https://smashthewalls.com/',
    'https://jacksonpollock.org/',
    'http://endless.horse/',
    'http://drawing.garden/',
    'https://www.trypap.com/',
    'http://www.republiquedesmangues.fr/',
    'http://www.movenowthinklater.com/',
    'https://sliding.toys/klotski/easy-street/',
    'https://paint.toys/calligram/',
    'https://checkboxrace.com/',
    'http://www.rrrgggbbb.com/',
    'http://www.koalastothemax.com/',
    'https://rotatingsandwiches.com/',
    'http://www.everydayim.com/',
    'http://randomcolour.com/',
    'http://maninthedark.com/',
    'http://cat-bounce.com/',
    'http://chrismckenzie.com/',
    'https://thezen.zone/',
    'http://ninjaflex.com/',
    'http://ihasabucket.com/',
    'http://corndogoncorndog.com/',
    'http://www.hackertyper.com/',
    'https://pointerpointer.com',
    'http://imaninja.com/',
    'http://www.partridgegetslucky.com/',
    'http://www.ismycomputeron.com/',
    'http://www.nullingthevoid.com/',
    'http://www.muchbetterthanthis.com/',
    'http://www.yesnoif.com/',
    'http://lacquerlacquer.com',
    'http://potatoortomato.com/',
    'http://iamawesome.com/',
    'https://strobe.cool/',
    'http://thisisnotajumpscare.com/',
    'http://doughnutkitten.com/',
    'http://crouton.net/',
    'http://corgiorgy.com/',
    'http://www.wutdafuk.com/',
    'http://unicodesnowmanforyou.com/',
    'http://chillestmonkey.com/',
    'http://scroll-o-meter.club/',
    'http://www.crossdivisions.com/',
    'http://tencents.info/',
    'https://memory.toys/maze/easy/',
    'https://boringboringboring.com/',
    'http://www.patience-is-a-virtue.org/',
    'http://pixelsfighting.com/',
    'http://isitwhite.com/',
    'https://existentialcrisis.com/',
    'http://onemillionlols.com/',
    'http://www.omfgdogs.com/',
    'http://oct82.com/',
    'http://chihuahuaspin.com/',
    'http://www.blankwindows.com/',
    'http://tunnelsnakes.com/',
    'http://www.trashloop.com/',
    'http://spaceis.cool/',
    'https://elonjump.com/',
    'http://www.doublepressure.com/',
    'http://www.donothingfor2minutes.com/',
    'http://buildshruggie.com/',
    // 'http://buzzybuzz.biz/', cert
    'http://yeahlemons.com/',
    'http://wowenwilsonquiz.com',
    // 'https://thepigeon.org/', expired
    'http://notdayoftheweek.com/',
    'http://www.amialright.com/',
    'https://optical.toys/thatcher-effect/',
    'https://greatbignothing.com/',
    'https://zoomquilt.org/',
    'https://dadlaughbutton.com/',
    'https://remoji.com/',
    'http://papertoilet.com/',
    'https://loopedforinfinity.com/',
    'https://end.city/',
    'https://www.bouncingdvdlogo.com/',
    'https://clicking.toys/peg-solitaire/english/',
    'https://toms.toys',
  ]

  var sites = null

  // Prepares and binds the button
  var init = function () {
    button.onclick = onButtonClick

    // Initial set sites
    sites = sitesList.slice()

    if (localStorage[sitelistName]) {
      // They have storage, filter out any not in the base list, that could be spam now
      var currentSites = JSON.parse(localStorage[sitelistName])
      var filteredSites = currentSites.filter(
        (site) => sitesList.indexOf(site) !== -1,
      )
      if (filteredSites.length > 0) {
        sites = filteredSites
      }
    }
  }

  // Selects and removes the next website from the list
  var selectWebsite = function () {
    var site, range, index

    range = randomRange > sites.length ? sites.length : randomRange
    index = Math.floor(Math.random() * range)

    site = sites[index]
    sites.splice(index, 1)

    return site
  }

  var openSite = function (url) {
    window.open(url)
  }

  var onButtonClick = function () {
    if (window.gtag) {
      gtag('event', 'click', { event_category: 'button' })
    }

    if (initialClick === false) {
      // Change text from "TO A"
      document.getElementById('joint').innerHTML = 'TO ANOTHER'
      initialClick = true
    }

    var url = selectWebsite()
    openSite(url)

    // User has visited ALL the sites... refresh the list.
    if (sites.length == 0) {
      sites = sitesList.slice()
    }

    localStorage[sitelistName] = JSON.stringify(sites)
  }

  init()
}

var uselessWebButton = new uselessWebButton(document.getElementById('button'))
